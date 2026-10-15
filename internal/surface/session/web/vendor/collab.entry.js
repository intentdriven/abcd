// The browser editor's third-party code, bundled once into collab.js.
export * as Y from "yjs";
export { Awareness, encodeAwarenessUpdate, applyAwarenessUpdate, removeAwarenessStates } from "y-protocols/awareness";
export { yCollab, yUndoManagerKeymap } from "y-codemirror.next";
export { EditorView, basicSetup } from "codemirror";
export { EditorState } from "@codemirror/state";
export { keymap } from "@codemirror/view";
export { markdown } from "@codemirror/lang-markdown";
export { default as qrcode } from "qrcode-generator";
