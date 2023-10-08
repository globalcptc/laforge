import { Component, OnInit } from '@angular/core';
import { TitleService } from '@services/title/title.service';

@Component({
  selector: 'laforge-layout',
  templateUrl: './layout.component.html',
  styleUrls: ['./layout.component.scss']
})
export class LayoutComponent implements OnInit {
  constructor(public title: TitleService) {}

  ngOnInit(): void {}

  getRandomInt(): number {
    return Math.floor(Math.random() * 2048);
  }
}
