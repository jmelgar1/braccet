import { Component } from '@angular/core';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';

@Component({
  selector: 'app-tournament-new',
  imports: [Breadcrumb],
  templateUrl: './tournament-new.html',
  styleUrl: './tournament-new.css'
})
export class TournamentNew {
  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Tournaments', route: '/tournaments' },
    { label: 'New Tournament' }
  ];
}
