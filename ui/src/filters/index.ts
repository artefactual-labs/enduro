import Vue from 'vue';

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
