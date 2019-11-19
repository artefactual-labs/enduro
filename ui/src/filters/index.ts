import Vue from 'vue';

Vue.filter('formatDateTime', (value: string) => {
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
