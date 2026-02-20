import { Routes } from '@angular/router';
import { Home } from './home/home';
import { Login } from './login/login';
import { Signup } from './signup/signup';
import { Tournaments } from './tournaments/tournaments';
import { TournamentNew } from './tournaments/new/tournament-new';

export const routes: Routes = [
  { path: '', component: Home },
  { path: 'login', component: Login },
  { path: 'signup', component: Signup },
  { path: 'tournaments', component: Tournaments },
  { path: 'tournaments/new', component: TournamentNew },
];
