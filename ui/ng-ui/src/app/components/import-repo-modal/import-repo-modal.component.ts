import { Component, Inject, OnInit } from '@angular/core';
import { FormControl, Validators } from '@angular/forms';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ApiService } from '@services/api/api.service';
import { BehaviorSubject } from 'rxjs';

@Component({
  selector: 'app-import-repo-modal',
  templateUrl: './import-repo-modal.component.html',
  styleUrls: ['./import-repo-modal.component.scss']
})
export class ImportRepoModalComponent implements OnInit {
  // Validate the gitUrl input is a github ssh url
  gitUrl = new FormControl('', [Validators.required, Validators.pattern('(.*?)@(.*?):(?:(.*?)/)?(.*?/.*?)')]);
  branchName = new FormControl('', Validators.required);
  envFilePath = new FormControl('', Validators.required);
  errorMessage: BehaviorSubject<string>;
  gitIsLoading: BehaviorSubject<boolean>;

  constructor(
    public dialogRef: MatDialogRef<ImportRepoModalComponent>,
    @Inject(MAT_DIALOG_DATA) public data: null,
    private api: ApiService,
    private snackbar: MatSnackBar
  ) {
    this.gitIsLoading = new BehaviorSubject(false);
  }

  ngOnInit(): void {
    this.errorMessage = new BehaviorSubject<string>('');
  }

  getGitUrlErrorMessage(): string {
    if (this.gitUrl.hasError('required')) {
      return 'This field is required';
    }
    if (this.gitUrl.hasError('pattern')) {
      return 'Git URL must be a SSH URL';
    }
    return '';
  }

  getBranchNameErrorMessage(): string {
    if (this.branchName.hasError('required')) {
      return 'This field is required';
    }
    return '';
  }

  getEnvFilePathErrorMessage(): string {
    if (this.envFilePath.hasError('required')) {
      return 'This field is required';
    }
    return '';
  }

  cancel() {
    this.dialogRef.close();
  }

  submit() {
    if (this.gitUrl.errors) return;
    if (this.branchName.errors) return;
    if (this.envFilePath.errors) return;
    this.gitIsLoading.next(true);
    this.snackbar.open('Importing...', null, {
      panelClass: ['bg-info', 'text-white']
    });
    this.api
      .createEnvFromGit({
        repoURL: this.gitUrl.value,
        branchName: this.branchName.value,
        envFilePath: this.envFilePath.value
      })
      .then(
        (env) => {
          if (env.length > 0) {
            this.snackbar.open('Environment successfully loaded. Refreshing page...', null, {
              panelClass: ['bg-success', 'text-white']
            });
            window.location.reload();
          }
        },
        (err) => {
          console.error(err);
          this.snackbar.open('Error while cloning repo from git. See console/logs for details.', 'Okay', {
            panelClass: ['bg-danger', 'text-white']
          });
        }
      )
      .finally(() => this.gitIsLoading.next(false));
  }
}
