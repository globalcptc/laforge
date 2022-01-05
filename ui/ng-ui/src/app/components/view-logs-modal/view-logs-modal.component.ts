import { Component, Inject, OnInit } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { LaForgeGetServerTasksQuery, LaForgeServerTaskType, LaForgeStreamServerTaskLogGQL } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { BehaviorSubject, Subscription } from 'rxjs';

type ParsedLog = { text: string; level: 'debug' | 'info' | 'warning' | 'error' | '' };
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
  parsedLogs: BehaviorSubject<ParsedLog[]>;
  parsedLogsLoading: BehaviorSubject<boolean>;

  constructor(
    public dialogRef: MatDialogRef<ViewLogsModalComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { taskUUIDs: string[] },
    private api: ApiService,
    private streamServerTaskLogsGQL: LaForgeStreamServerTaskLogGQL
  ) {
    this.serverTasks = new BehaviorSubject([]);
    this.logText = new BehaviorSubject('');
    this.showLog = new BehaviorSubject(false);
    this.parsedLogs = new BehaviorSubject([]);
    this.parsedLogsLoading = new BehaviorSubject(false);
  }

  ngOnInit(): void {
    this.api
      .getServerTasks(this.data.taskUUIDs)
      .then((serverTasks) =>
        this.serverTasks.next([...serverTasks].sort((a, b) => new Date(b.start_time).getTime() - new Date(a.start_time).getTime()))
      );
  }

  onClose(): void {
    if (this.logSubscription) this.logSubscription.unsubscribe();
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
        this.parsedLogsLoading.next(true);
        // this.logText.next(streamServerTaskLog);
        const newParsedLogs: ParsedLog[] = [];
        let prevLine: ParsedLog = null;
        for (const line of streamServerTaskLog.split('\n')) {
          let level: ParsedLog['level'] = '';
          if (line.indexOf('level=debug') >= 0) level = 'debug';
          if (line.indexOf('level=info') >= 0) level = 'info';
          if (line.indexOf('level=warning') >= 0) level = 'warning';
          if (line.indexOf('level=error') >= 0) level = 'error';
          if (!prevLine) {
            prevLine = { text: line, level };
            continue;
          } else if (prevLine.level === level) {
            prevLine = { text: prevLine.text + '\n' + line, level };
            continue;
          } else {
            newParsedLogs.push({ ...prevLine });
            prevLine = { text: line, level };
          }
        }
        this.parsedLogsLoading.next(false);
        this.parsedLogs.next(newParsedLogs);
      });
  }

  getLevelColor(log) {
    switch (log.level) {
      case 'debug':
        return 'white';
      case 'info':
        return 'info';
      case 'warning':
        return 'warning';
      case 'error':
        return 'danger';
      default:
        return 'white';
    }
  }

  clearLog() {
    this.logSubscription.unsubscribe();
    this.logSubscription = null;
    // this.logText.next('');
    this.parsedLogs.next([]);
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
