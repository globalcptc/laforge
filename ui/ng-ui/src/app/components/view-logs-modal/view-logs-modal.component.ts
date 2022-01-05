import { Component, Inject, OnInit } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { LaForgeGetServerTasksQuery, LaForgeServerTaskType, LaForgeStreamServerTaskLogGQL } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { BehaviorSubject, Subscription } from 'rxjs';

@Component({
  selector: 'app-view-logs-modal',
  templateUrl: './view-logs-modal.component.html',
  styleUrls: ['./view-logs-modal.component.scss']
})
export class ViewLogsModalComponent implements OnInit {
  serverTasks: BehaviorSubject<LaForgeGetServerTasksQuery['serverTasks']>;
  logText: BehaviorSubject<string>;
  showLog: BehaviorSubject<boolean>;
  logSubscription: Subscription;

  constructor(
    public dialogRef: MatDialogRef<ViewLogsModalComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { taskUUIDs: string[] },
    private api: ApiService,
    private streamServerTaskLogsGQL: LaForgeStreamServerTaskLogGQL
  ) {
    this.serverTasks = new BehaviorSubject([]);
    this.logText = new BehaviorSubject('');
    this.showLog = new BehaviorSubject(false);
  }

  ngOnInit(): void {
    this.api
      .getServerTasks(this.data.taskUUIDs)
      .then((serverTasks) =>
        this.serverTasks.next([...serverTasks].sort((a, b) => new Date(b.start_time).getTime() - new Date(a.start_time).getTime()))
      );
  }

  onClose(): void {
    this.dialogRef.close();
  }

  viewLog(serverTask: LaForgeGetServerTasksQuery['serverTasks'][0]) {
    this.showLog.next(true);
    this.logSubscription = this.streamServerTaskLogsGQL
      .subscribe({
        taskUUID: serverTask.id
      })
      .subscribe(({ data: { streamServerTaskLog }, errors }) => {
        if (errors) {
          return this.logText.error(errors);
        }
        this.logText.next(streamServerTaskLog);
      });
  }

  clearLog() {
    this.logSubscription.unsubscribe();
    this.logText.next('');
    this.showLog.next(false);
  }

  getServerTaskColor(task: LaForgeGetServerTasksQuery['serverTasks'][0]) {
    switch (task.type) {
      case LaForgeServerTaskType.Createbuild:
        return 'primary';
      case LaForgeServerTaskType.Deletebuild:
        return 'danger';
      case LaForgeServerTaskType.Executebuild:
        return 'success';
      case LaForgeServerTaskType.Loadenv:
        return 'info';
      case LaForgeServerTaskType.Rebuild:
        return 'warning';
      case LaForgeServerTaskType.Renderfiles:
        return 'primary';
      default:
        return 'dark';
    }
  }
}
