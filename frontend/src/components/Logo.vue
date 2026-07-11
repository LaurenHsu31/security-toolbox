<script setup lang="ts">
// The brand mark: a hex nut (toolbox) with a keyhole (security).
// Inline SVG so it inherits no external requests and scales crisply.
withDefaults(defineProps<{ size?: number }>(), { size: 22 })
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 32 32"
    fill="none"
    role="img"
    aria-label="security-toolbox logo"
  >
    <defs>
      <linearGradient id="stb-g" x1="6" y1="4" x2="27" y2="29" gradientUnits="userSpaceOnUse">
        <stop offset="0" stop-color="#0a84ff" />
        <stop offset="1" stop-color="#0057c2" />
      </linearGradient>
    </defs>
    <!-- Hex nut body; the wide round stroke gives the corners their radius. -->
    <path
      d="M16 3.6 L26.75 9.8 V22.2 L16 28.4 L5.25 22.2 V9.8 Z"
      fill="url(#stb-g)"
      stroke="url(#stb-g)"
      stroke-width="3.2"
      stroke-linejoin="round"
    />
    <!-- Keyhole -->
    <circle cx="16" cy="13.1" r="3.5" fill="#fff" />
    <path
      d="M14.9 15.9 L14 21.2 H18 L17.1 15.9 Z"
      fill="#fff"
      stroke="#fff"
      stroke-width="1.8"
      stroke-linejoin="round"
    />
  </svg>
</template>
