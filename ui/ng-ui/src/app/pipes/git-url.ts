import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ name: 'gitUrl' })
export class GitUrlPipe implements PipeTransform {
  transform(hash: string, sshUrl: string): string {
    const provider = sshUrl.split(':')[0].split('@')[1];
    const repoPath = sshUrl.split(':')[1].replace('.git', '');
    switch (provider) {
      case 'github.com':
        return `https://github.com/${repoPath}/commit/${hash}`;
      default:
        return '#';
    }
  }
}
