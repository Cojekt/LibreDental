/// <reference types="svelte" />
/// <reference types="vite/client" />

declare module "*bindings/*.js" {
  const content: any;
  export default content;
}
