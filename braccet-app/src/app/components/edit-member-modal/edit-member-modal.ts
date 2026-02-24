import { Component, input, output, signal, inject, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ImageCropperComponent, ImageCroppedEvent, ImageTransform } from 'ngx-image-cropper';
import { CommunityMember, UpdateMemberRequest } from '../../models/community.model';
import { CommunityService } from '../../services/community.service';
import { UploadService } from '../../services/upload.service';
import { switchMap } from 'rxjs';

@Component({
  selector: 'app-edit-member-modal',
  imports: [FormsModule, ImageCropperComponent],
  templateUrl: './edit-member-modal.html',
  styleUrl: './edit-member-modal.css'
})
export class EditMemberModal {
  private communityService = inject(CommunityService);
  private uploadService = inject(UploadService);

  member = input.required<CommunityMember>();
  communitySlug = input.required<string>();

  close = output<void>();
  memberUpdated = output<CommunityMember>();
  requestPromotion = output<void>();

  // Tracks member ID after promotion (for participants that start with id=0)
  promotedMemberId = signal<number | null>(null);
  private pendingAction: 'crop' | 'save' | null = null;

  displayName = signal('');
  iconPreview = signal<string | null>(null);
  selectedFile = signal<File | null>(null);
  saving = signal(false);
  error = signal('');

  // Cropper state
  showCropper = signal(false);
  imageTransform = signal<ImageTransform>({ scale: 1 });
  croppingInProgress = signal(false);
  private croppedBlob: Blob | null = null;

  // Region state
  region = signal<string>('');
  availableRegions = signal<string[]>([]);
  showAddRegion = signal(false);
  newRegionInput = signal('');
  loadingRegions = signal(false);

  constructor() {
    // Initialize from member
    effect(() => {
      const m = this.member();
      this.displayName.set(m.display_name);
      this.region.set(m.region || '');
      if (m.icon_url) {
        this.iconPreview.set(m.icon_url);
      }
    });

    // Fetch available regions when communitySlug is set
    effect(() => {
      const slug = this.communitySlug();
      if (slug) {
        this.loadRegions(slug);
      }
    });
  }

  private loadRegions(slug: string): void {
    this.loadingRegions.set(true);
    this.communityService.getRegions(slug).subscribe({
      next: (regions) => {
        // Ensure the member's current region is included in the list
        const currentRegion = this.member().region;
        if (currentRegion && !regions.includes(currentRegion)) {
          regions = [...regions, currentRegion].sort();
        }
        this.availableRegions.set(regions);
        this.loadingRegions.set(false);
      },
      error: () => {
        // On error, at least include the current region if it exists
        const currentRegion = this.member().region;
        if (currentRegion) {
          this.availableRegions.set([currentRegion]);
        }
        this.loadingRegions.set(false);
      }
    });
  }

  onRegionSelectChange(value: string): void {
    if (value === '__new__') {
      this.showAddRegion.set(true);
      this.newRegionInput.set('');
    } else {
      this.showAddRegion.set(false);
      this.region.set(value);
    }
  }

  onNewRegionInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    // Convert to uppercase and limit to 2 chars
    const value = input.value.toUpperCase().replace(/[^A-Z]/g, '').slice(0, 2);
    this.newRegionInput.set(value);
    input.value = value;
  }

  confirmNewRegion(): void {
    const newRegion = this.newRegionInput();
    if (newRegion.length === 2) {
      this.region.set(newRegion);
      // Add to available regions if not already present
      if (!this.availableRegions().includes(newRegion)) {
        this.availableRegions.set([...this.availableRegions(), newRegion].sort());
      }
      this.showAddRegion.set(false);
    }
  }

  cancelNewRegion(): void {
    this.showAddRegion.set(false);
    this.newRegionInput.set('');
  }

  // Get effective member ID (promoted ID takes precedence over input member ID)
  private getMemberId(): number {
    return this.promotedMemberId() ?? this.member().id;
  }

  // Called by parent after promotion completes
  completePromotion(memberId: number): void {
    this.promotedMemberId.set(memberId);
    // Continue pending action
    if (this.pendingAction === 'crop') {
      this.pendingAction = null;
      this.performCrop();
    } else if (this.pendingAction === 'save') {
      this.pendingAction = null;
      this.performSave();
    }
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    const validation = this.uploadService.validateImageFile(file);

    if (!validation.valid) {
      this.error.set(validation.error || 'Invalid file');
      return;
    }

    this.error.set('');
    this.selectedFile.set(file);

    // Create preview
    const reader = new FileReader();
    reader.onload = (e) => {
      this.iconPreview.set(e.target?.result as string);
    };
    reader.readAsDataURL(file);
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();

    const files = event.dataTransfer?.files;
    if (!files || files.length === 0) return;

    const file = files[0];
    const validation = this.uploadService.validateImageFile(file);

    if (!validation.valid) {
      this.error.set(validation.error || 'Invalid file');
      return;
    }

    this.error.set('');
    this.selectedFile.set(file);

    // Create preview
    const reader = new FileReader();
    reader.onload = (e) => {
      this.iconPreview.set(e.target?.result as string);
    };
    reader.readAsDataURL(file);
  }

  removeIcon(): void {
    this.selectedFile.set(null);
    this.iconPreview.set(null);
  }

  // Check if selected file is SVG (SVGs cannot be cropped)
  isSvgFile(): boolean {
    return this.selectedFile()?.type === 'image/svg+xml';
  }

  // Cropper methods
  openCropper(): void {
    if (!this.iconPreview() || this.isSvgFile()) return;
    this.imageTransform.set({ scale: 1 });
    this.showCropper.set(true);
  }

  cancelCrop(): void {
    this.showCropper.set(false);
    this.croppedBlob = null;
  }

  zoomIn(): void {
    const current = this.imageTransform();
    this.imageTransform.set({ ...current, scale: Math.min((current.scale || 1) + 0.1, 3) });
  }

  zoomOut(): void {
    const current = this.imageTransform();
    this.imageTransform.set({ ...current, scale: Math.max((current.scale || 1) - 0.1, 0.5) });
  }

  onImageCropped(event: ImageCroppedEvent): void {
    if (event.blob) {
      this.croppedBlob = event.blob;
    }
  }

  applyCrop(): void {
    if (!this.croppedBlob) return;

    // If member doesn't exist yet (id=0), request promotion first
    if (this.getMemberId() === 0) {
      this.pendingAction = 'crop';
      this.requestPromotion.emit();
      return;
    }

    this.performCrop();
  }

  private performCrop(): void {
    if (!this.croppedBlob) return;

    this.croppingInProgress.set(true);

    const file = new File([this.croppedBlob], 'cropped-icon.png', { type: 'image/png' });
    const slug = this.communitySlug();
    const memberId = this.getMemberId();

    this.communityService.getMemberIconUploadUrl(slug, memberId, file.type)
      .pipe(
        switchMap(response => {
          return this.uploadService.uploadToPresignedUrl(response.upload_url, file).pipe(
            switchMap(() => {
              return this.communityService.updateMember(slug, memberId, {
                icon_url: response.icon_url
              });
            })
          );
        })
      )
      .subscribe({
        next: (updatedMember) => {
          // Update preview with cache buster
          this.iconPreview.set(updatedMember.icon_url + '?t=' + Date.now());
          this.showCropper.set(false);
          this.croppingInProgress.set(false);
          this.selectedFile.set(null);
          this.croppedBlob = null;
          this.memberUpdated.emit(updatedMember);
        },
        error: (err) => {
          this.croppingInProgress.set(false);
          this.error.set(err.message || 'Failed to upload cropped image');
        }
      });
  }

  canSave(): boolean {
    const name = this.displayName().trim();
    return name.length > 0 && !this.saving();
  }

  save(): void {
    const newDisplayName = this.displayName().trim();

    if (!newDisplayName) {
      this.error.set('Display name is required');
      return;
    }

    // If member doesn't exist yet (id=0), request promotion first
    if (this.getMemberId() === 0) {
      this.pendingAction = 'save';
      this.requestPromotion.emit();
      return;
    }

    this.performSave();
  }

  private performSave(): void {
    const file = this.selectedFile();
    const slug = this.communitySlug();
    const memberId = this.getMemberId();
    const newDisplayName = this.displayName().trim();

    this.saving.set(true);
    this.error.set('');

    // If there's a new file, upload it first
    if (file) {
      const currentRegion = this.region();
      const originalRegion = this.member().region || '';

      this.communityService.getMemberIconUploadUrl(slug, memberId, file.type)
        .pipe(
          switchMap(response => {
            return this.uploadService.uploadToPresignedUrl(response.upload_url, file).pipe(
              switchMap(() => {
                // After upload, update member with new icon URL, display name, and region
                const updateRequest: UpdateMemberRequest = {
                  display_name: newDisplayName,
                  icon_url: response.icon_url
                };
                if (currentRegion !== originalRegion) {
                  updateRequest.region = currentRegion;
                }
                return this.communityService.updateMember(slug, memberId, updateRequest);
              })
            );
          })
        )
        .subscribe({
          next: (updatedMember) => {
            this.saving.set(false);
            this.memberUpdated.emit(updatedMember);
            this.close.emit();
          },
          error: (err) => {
            this.saving.set(false);
            this.error.set(err.message || 'Failed to save changes');
          }
        });
    } else {
      // No new file, just update display name, region (and possibly clear icon)
      const updateRequest: UpdateMemberRequest = {
        display_name: newDisplayName
      };

      // If icon was removed (preview is null but member had an icon)
      if (this.iconPreview() === null && this.member().icon_url) {
        updateRequest.icon_url = '';
      }

      // Include region if changed
      const currentRegion = this.region();
      const originalRegion = this.member().region || '';
      if (currentRegion !== originalRegion) {
        updateRequest.region = currentRegion;
      }

      this.communityService.updateMember(slug, memberId, updateRequest)
        .subscribe({
          next: (updatedMember) => {
            this.saving.set(false);
            this.memberUpdated.emit(updatedMember);
            this.close.emit();
          },
          error: (err) => {
            this.saving.set(false);
            this.error.set(err.error?.error || 'Failed to save changes');
          }
        });
    }
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.close.emit();
    }
  }
}
