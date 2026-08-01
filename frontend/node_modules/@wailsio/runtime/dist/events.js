/*
 _	   __	  _ __
| |	 / /___ _(_) /____
| | /| / / __ `/ / / ___/
| |/ |/ / /_/ / / (__  )
|__/|__/\__,_/_/_/____/
The electron alternative for Go
(c) Lea Anthony 2019-present
*/
import { newRuntimeCaller, objectNames } from "./runtime.js";
import { eventListeners, Listener, listenerOff } from "./listener.js";
import { Events as Create } from "./create.js";
import { Types } from "./event_types.js";
// Setup
import { hasDOM } from "./environment.js";
if (hasDOM) {
    window._wails = window._wails || {};
    window._wails.dispatchWailsEvent = dispatchWailsEvent;
}
const call = newRuntimeCaller(objectNames.Events);
const EmitMethod = 0;
export * from "./event_types.js";
/**
 * Represents a system event or a custom event emitted through wails-provided facilities.
 */
export class WailsEvent {
    constructor(name, data) {
        this.name = name;
        this.data = data !== null && data !== void 0 ? data : null;
    }
}
function dispatchWailsEvent(event) {
    let listeners = eventListeners.get(event.name);
    if (!listeners) {
        return;
    }
    let wailsEvent = new WailsEvent(event.name, (event.name in Create) ? Create[event.name](event.data) : event.data);
    if ('sender' in event) {
        wailsEvent.sender = event.sender;
    }
    // Dispatch to a snapshot, then remove all expired listeners in a single
    // post-dispatch filter of the live map.
    // - Writing the snapshot back wholesale would undo subscription changes
    //   made inside a handler (#4393).
    // - Calling listenerOff() per expired listener inside the loop is O(n²)
    //   when many listeners expire on the same event.
    // Filtering the live array once after dispatch is O(n) and still honours
    // any listeners added or removed by handlers during dispatch.
    const expired = new Set();
    for (const listener of listeners.slice()) {
        if (listener.dispatch(wailsEvent)) {
            expired.add(listener);
        }
    }
    if (expired.size > 0) {
        const live = eventListeners.get(event.name);
        if (live) {
            const remaining = live.filter(l => !expired.has(l));
            if (remaining.length === 0) {
                eventListeners.delete(event.name);
            }
            else {
                eventListeners.set(event.name, remaining);
            }
        }
    }
}
/**
 * Register a callback function to be called multiple times for a specific event.
 *
 * @param eventName - The name of the event to register the callback for.
 * @param callback - The callback function to be called when the event is triggered.
 * @param maxCallbacks - The maximum number of times the callback can be called for the event. Once the maximum number is reached, the callback will no longer be called.
 * @returns A function that, when called, will unregister the callback from the event.
 */
export function OnMultiple(eventName, callback, maxCallbacks) {
    let listeners = eventListeners.get(eventName) || [];
    const thisListener = new Listener(eventName, callback, maxCallbacks);
    listeners.push(thisListener);
    eventListeners.set(eventName, listeners);
    return () => listenerOff(thisListener);
}
/**
 * Registers a callback function to be executed when the specified event occurs.
 *
 * @param eventName - The name of the event to register the callback for.
 * @param callback - The callback function to be called when the event is triggered.
 * @returns A function that, when called, will unregister the callback from the event.
 */
export function On(eventName, callback) {
    return OnMultiple(eventName, callback, -1);
}
/**
 * Registers a callback function to be executed only once for the specified event.
 *
 * @param eventName - The name of the event to register the callback for.
 * @param callback - The callback function to be called when the event is triggered.
 * @returns A function that, when called, will unregister the callback from the event.
 */
export function Once(eventName, callback) {
    return OnMultiple(eventName, callback, 1);
}
/**
 * Removes event listeners for the specified event names.
 *
 * @param eventNames - The name of the events to remove listeners for.
 */
export function Off(...eventNames) {
    eventNames.forEach(eventName => eventListeners.delete(eventName));
}
/**
 * Removes all event listeners.
 */
export function OffAll() {
    eventListeners.clear();
}
export function Emit(name, data) {
    return call(EmitMethod, new WailsEvent(name, data));
}
