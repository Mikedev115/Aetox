// Is a side-panel handle being dragged right now?
//
// One flag, set by App.svelte's resize handles and read by the native browser
// pane. The pane needs it because it is not a DOM element: it is a real OS
// window re-glued to its box with a native call on every ResizeObserver tick,
// and a page relaid out sixty times a second is what made a drag look rough.
// With the flag the pane holds still for the length of the drag and follows
// once, when the handle is let go (BrowserPane.svelte, reflow).
//
// A module of its own rather than a field on cockpit: the fact is the layout's,
// not the chat's, and the two files that care should not have to meet in the
// store that holds everything else.
export const panelDrag = $state({ active: false })
