/// <reference types="astro/client" />

declare module '@alpinejs/csp' {
  const Alpine: typeof import('alpinejs').default;
  export default Alpine;
}
