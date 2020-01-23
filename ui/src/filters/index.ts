import Vue from 'vue';
import moment from 'moment';
import humanizeDuration from 'humanize-duration';

Vue.filter('formatDateTimeString', (value: string) => {
  const date = new Date(value);
  return date.toLocaleString();
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

Vue.filter('formatEpochDuration', (from: number, to: number) => {
  const f = new Date(Math.round(from / 1000 / 1000));
  const t = new Date(Math.round(to / 1000 / 1000));
  const diff = moment(t).diff(f);
  return humanizeDuration(moment.duration(diff).asMilliseconds());
});
