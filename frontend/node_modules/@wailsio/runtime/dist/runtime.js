/*
 _     __     _ __
| |  / /___ _(_) /____
| | /| / / __ `/ / / ___/
| |/ |/ / /_/ / / (__  )
|__/|__/\__,_/_/_/____/
The electron alternative for Go
(c) Lea Anthony 2019-present
*/
var _a;
import { nanoid } from "./nanoid.js";
import { hasDOM } from "./environment.js";
// Resolved lazily: window does not exist when the module is imported during
// server-side rendering (#4679), and nothing can call the runtime there.
function runtimeURL() {
    return window.location.origin + "/wails/runtime";
}
// Stay under WebView2's ~2MB request body buffering limit in WebResourceRequested.
const CHUNK_THRESHOLD = 512 * 1024;
// Re-export nanoid for custom transport implementations
export { nanoid };
/**
 * Exception class that will be thrown in case the bound method returns an error.
 * The value of the {@link RuntimeError#name} property is "RuntimeError".
 */
export class RuntimeError extends Error {
    /**
     * Constructs a new RuntimeError instance.
     * @param message - The error message.
     * @param options - Options to be forwarded to the Error constructor.
     */
    constructor(message, options) {
        super(message, options);
        this.name = "RuntimeError";
    }
}
// Object Names
export const objectNames = Object.freeze({
    Call: 0,
    Clipboard: 1,
    Application: 2,
    Events: 3,
    ContextMenu: 4,
    Dialog: 5,
    Window: 6,
    Screens: 7,
    System: 8,
    Browser: 9,
    CancelCall: 10,
    IOS: 11,
    Android: 12,
});
export let clientId = nanoid();
/**
 * Custom transport implementation (can be set by user)
 */
let customTransport = null;
/**
 * Set a custom transport for all Wails runtime calls.
 * This allows you to replace the default HTTP fetch transport with
 * WebSockets, custom protocols, or any other mechanism.
 *
 * @param transport - Your custom transport implementation
 *
 * @example
 * ```typescript
 * import { setTransport } from '/wails/runtime.js';
 *
 * const wsTransport = {
 *   call: async (objectID, method, windowName, args) => {
 *     // Your WebSocket implementation
 *   }
 * };
 *
 * setTransport(wsTransport);
 * ```
 */
export function setTransport(transport) {
    customTransport = transport;
}
/**
 * Get the current transport (useful for extending/wrapping)
 */
export function getTransport() {
    return customTransport;
}
/**
 * Creates a new runtime caller with specified ID.
 *
 * @param object - The object to invoke the method on.
 * @param windowName - The name of the window.
 * @return The new runtime caller function.
 */
export function newRuntimeCaller(object, windowName = '') {
    return function (method, args = null) {
        return runtimeCallWithID(object, method, windowName, args);
    };
}
async function runtimeCallWithID(objectID, method, windowName, args) {
    var _a, _b;
    // Use custom transport if available
    if (customTransport) {
        return customTransport.call(objectID, method, windowName, args);
    }
    // Default HTTP fetch transport
    let url = new URL(runtimeURL());
    let body = {
        object: objectID,
        method
    };
    if (args !== null && args !== undefined) {
        body.args = args;
    }
    let headers = {
        ["x-wails-client-id"]: clientId,
        ["Content-Type"]: "application/json"
    };
    if (windowName) {
        headers["x-wails-window-name"] = windowName;
    }
    const bodyStr = JSON.stringify(body);
    let response;
    if (bodyStr.length > CHUNK_THRESHOLD) {
        response = await sendChunked(url, headers, bodyStr);
    }
    else {
        response = await fetch(url, { method: 'POST', headers, body: bodyStr });
    }
    if (!response.ok) {
        const ct = response.headers.get("Content-Type");
        if (ct === null || ct === void 0 ? void 0 : ct.includes("application/json")) {
            const json = await response.json();
            let err;
            switch (json.kind) {
                case "ReferenceError":
                    err = new ReferenceError(json.message);
                    break;
                case "TypeError":
                    err = new TypeError(json.message);
                    break;
                case "RuntimeError":
                    err = new RuntimeError(json.message);
                    break;
                default: err = new Error(json.message);
            }
            err.cause = json.cause;
            throw err;
        }
        throw new Error(await response.text());
    }
    if (((_b = (_a = response.headers.get("Content-Type")) === null || _a === void 0 ? void 0 : _a.indexOf("application/json")) !== null && _b !== void 0 ? _b : -1) !== -1) {
        return response.json();
    }
    else {
        return response.text();
    }
}
// sendChunked splits a large serialised request body into CHUNK_THRESHOLD-sized
// byte chunks and sends them serially.  Encoding to UTF-8 bytes before slicing
// prevents corruption of non-BMP characters (surrogate pairs) that would occur
// when splitting at JavaScript string indices.  The Go transport assembles the
// raw bytes before processing.  Only the final chunk's response carries the RPC result.
async function sendChunked(url, headers, bodyStr) {
    const chunkId = nanoid();
    const bodyBytes = new TextEncoder().encode(bodyStr);
    const totalChunks = Math.ceil(bodyBytes.length / CHUNK_THRESHOLD);
    for (let i = 0; i < totalChunks - 1; i++) {
        const chunk = bodyBytes.subarray(i * CHUNK_THRESHOLD, (i + 1) * CHUNK_THRESHOLD);
        const resp = await fetch(url, {
            method: 'POST',
            headers: Object.assign(Object.assign({}, headers), { 'x-wails-chunk-id': chunkId, 'x-wails-chunk-index': String(i), 'x-wails-chunk-total': String(totalChunks) }),
            body: chunk,
        });
        if (!resp.ok) {
            throw new Error(await resp.text());
        }
    }
    return fetch(url, {
        method: 'POST',
        headers: Object.assign(Object.assign({}, headers), { 'x-wails-chunk-id': chunkId, 'x-wails-chunk-index': String(totalChunks - 1), 'x-wails-chunk-total': String(totalChunks) }),
        body: bodyBytes.subarray((totalChunks - 1) * CHUNK_THRESHOLD),
    });
}
const androidBridge = hasDOM &&
    typeof ((_a = window.wails) === null || _a === void 0 ? void 0 : _a.invokeAsync) === "function" ? window.wails : null;
if (androidBridge) {
    const pending = new Map();
    window._wailsAndroidCallback = (id, response, error) => {
        var _a;
        const promise = pending.get(id);
        if (!promise)
            return;
        pending.delete(id);
        if (error) {
            promise.reject(new Error(error));
            return;
        }
        try {
            const envelope = JSON.parse(response !== null && response !== void 0 ? response : "{}");
            if (!envelope.ok) {
                promise.reject(new Error((_a = envelope.error) !== null && _a !== void 0 ? _a : "unknown runtime call error"));
                return;
            }
            promise.resolve("text" in envelope ? envelope.text : envelope.data);
        }
        catch (e) {
            promise.reject(e);
        }
    };
    customTransport = {
        call(objectID, method, windowName, args) {
            return new Promise((resolve, reject) => {
                const id = nanoid();
                pending.set(id, { resolve, reject });
                try {
                    androidBridge.invokeAsync(id, JSON.stringify({
                        object: objectID,
                        method: method,
                        windowName: windowName,
                        args: args !== null && args !== void 0 ? args : null,
                        clientId: clientId,
                    }));
                }
                catch (e) {
                    // Don't leak the pending entry if dispatch throws synchronously
                    pending.delete(id);
                    reject(e);
                }
            });
        },
    };
}
