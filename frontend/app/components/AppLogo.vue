<template>
  <span class="app-logo">
    <span class="app-logo-mark">
      <UIcon
        name="entypo:flow-cascade"
        class="app-logo-icon"
      />
    </span>
    <span class="app-logo-text">Enduro</span>
  </span>
</template>

<style scoped>
.app-logo {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
}

.app-logo-mark {
  align-items: center;
  background: var(--app-brand-strong, var(--color-brand-600, #9e3d85));
  border-radius: 9999px;
  display: inline-flex;
  height: 2rem;
  justify-content: center;
  width: 2rem;
  flex-shrink: 0;
}

.app-logo-icon {
  color: var(--app-on-brand, #fff);
  font-size: 1.28rem;
  flex-shrink: 0;
}

.app-logo-text {
  color: var(--app-brand-strong, var(--color-brand-600, #9e3d85));
  font-size: 1.125rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1;
}
</style>
