import { Component, OnInit } from '@angular/core';
import { AuthService } from '@services/auth/auth.service';

@Component({
  selector: 'laforge-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit {
  constructor(private auth: AuthService) {}

  currentUser() {
    return this.auth.CurrentUser();
  }

  ngOnInit(): void {}
}
