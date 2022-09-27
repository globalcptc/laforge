import { Component } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';
import { ApiService } from '@services/api/api.service';
import { BehaviorSubject } from 'rxjs';

@Component({
  selector: 'app-nuke-db-modal',
  templateUrl: './nuke-db-modal.component.html',
  styleUrls: ['./nuke-db-modal.component.scss']
})
export class NukeDbModalComponent {
  nukeDbConfirmed: BehaviorSubject<boolean>;
  nukeDbLoading: BehaviorSubject<boolean>;

  constructor(
    public dialogRef: MatDialogRef<NukeDbModalComponent>,
    // @Inject(MAT_DIALOG_DATA) public data: { buildName: string; buildId: string },
    private api: ApiService,
    private router: Router,
    private snackbar: MatSnackBar
  ) {
    this.nukeDbConfirmed = new BehaviorSubject(false);
    this.nukeDbLoading = new BehaviorSubject(false);
  }

  yesIDoChange(value: string) {
    if (value.toLocaleLowerCase() === 'yes i really really do') {
      this.nukeDbConfirmed.next(true);
    } else {
      this.nukeDbConfirmed.next(false);
    }
  }

  onClose(): void {
    this.dialogRef.close();
  }

  triggerNukeDb(): void {
    this.nukeDbLoading.next(true);
    this.api
      .nukeBackend()
      .then(console.log, (err) => {
        console.error(err);
        this.snackbar.open('Error while wiping database. See console for more info.', 'Okay.', {
          panelClass: ['bg-danger', 'text-white']
        });
      })
      .finally(() => this.nukeDbLoading.next(false));
  }
}
