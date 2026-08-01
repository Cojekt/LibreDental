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
const call = newRuntimeCaller(objectNames.Screens);
const getAll = 0;
const getPrimary = 1;
const getCurrent = 2;
const getByID = 3;
const getByIndex = 4;
/**
 * Gets all screens.
 *
 * @returns A promise that resolves to an array of Screen objects.
 */
export function GetAll() {
    return call(getAll);
}
/**
 * Gets the primary screen.
 *
 * @returns A promise that resolves to the primary screen.
 */
export function GetPrimary() {
    return call(getPrimary);
}
/**
 * Gets the current active screen.
 *
 * @returns A promise that resolves with the current active screen.
 */
export function GetCurrent() {
    return call(getCurrent);
}
/**
 * Gets a screen by its unique display ID.
 *
 * @param id - The unique identifier of the screen.
 * @returns A promise that resolves to the matching Screen.
 */
export function GetByID(id) {
    return call(getByID, { id });
}
/**
 * Gets a screen by its index in the screen list.
 *
 * @param index - The zero-based index of the screen.
 * @returns A promise that resolves to the matching Screen.
 */
export function GetByIndex(index) {
    return call(getByIndex, { index });
}
