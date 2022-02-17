import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ name: 'gitHash' })
export class GitHashPipe implements PipeTransform {
  transform(value: string): string {
    return value.substring(0, 7);
  }
}
