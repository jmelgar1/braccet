import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class UploadService {
  /**
   * Upload a file directly to a presigned URL (e.g., R2/S3)
   * Uses XMLHttpRequest for direct PUT to cloud storage
   */
  uploadToPresignedUrl(url: string, file: File): Observable<void> {
    return new Observable(observer => {
      const xhr = new XMLHttpRequest();
      xhr.open('PUT', url, true);
      xhr.setRequestHeader('Content-Type', file.type);

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          observer.next();
          observer.complete();
        } else {
          observer.error(new Error(`Upload failed with status ${xhr.status}`));
        }
      };

      xhr.onerror = () => {
        observer.error(new Error('Upload failed due to network error'));
      };

      xhr.send(file);
    });
  }

  /**
   * Validate that a file is an allowed image type
   */
  validateImageFile(file: File): { valid: boolean; error?: string } {
    const allowedTypes = ['image/jpeg', 'image/png', 'image/webp'];
    const maxSize = 2 * 1024 * 1024; // 2MB

    if (!allowedTypes.includes(file.type)) {
      return { valid: false, error: 'File must be JPEG, PNG, or WebP' };
    }

    if (file.size > maxSize) {
      return { valid: false, error: 'File must be smaller than 2MB' };
    }

    return { valid: true };
  }
}
