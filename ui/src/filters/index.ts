import Vue from 'vue';
import moment from 'moment';
import humanizeDuration from 'humanize-duration';

export interface EventTime {
  seconds: number
  nanos: number
}

Vue.filter('formatDateTimeString', (value: string | EventTime): Date => {
    return typeof(value) === 'string'
        ? new Date(value)
        : fromEventTimeToDate(value);
});

Vue.filter('formatDateTime', (value: Date) => {
  if (!value) {
    return '';
  }
  return value.toLocaleString();
});

Vue.filter('formatEpoch', (value: number) => {
  if (!value) {
    return '';
  }
  const date = new Date(value / 1000 / 1000); // TODO: is this right?
  return date.toLocaleString();
});

Vue.filter('formatDuration', (from: Date, to: Date) => {
  const diff = moment(to).diff(from);
  return humanizeDuration(moment.duration(diff).asMilliseconds());
});

Vue.filter('formatEpochDuration', (from: EventTime, to: EventTime) => {
  const f = fromEventTimeToDate(from)
  const t = fromEventTimeToDate(to)
  const diff = moment(t).diff(f);
  return humanizeDuration(moment.duration(diff).asMilliseconds());
});

export function fromEventTimeToDate(ev :EventTime): Date {
  let secondsToMilis = ev.seconds * 1000;
  let nanosecondsToMilis = ev.nanos / 1_000_000;
  return new Date(secondsToMilis + nanosecondsToMilis);
}
