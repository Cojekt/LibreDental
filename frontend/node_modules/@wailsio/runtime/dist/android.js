/*
 _     __     _ __
| |  / /___ _(_) /____
| | /| / / __ `/ / / ___/
| |/ |/ / /_/ / / (__  )
|__/|__/\__,_/_/_/____/
The electron alternative for Go
(c) Lea Anthony 2019-present
*/
import { newRuntimeCaller, objectNames } from "./runtime.js";
const call = newRuntimeCaller(objectNames.Android);
// Method IDs (must match messageprocessor_android.go)
const HapticsVibrate = 0;
const DeviceInfo = 1;
const ToastShow = 2;
export var Haptics;
(function (Haptics) {
    /** Vibrate the device for the given duration in milliseconds. */
    function Vibrate(durationMs = 100) {
        return call(HapticsVibrate, { duration: durationMs });
    }
    Haptics.Vibrate = Vibrate;
})(Haptics || (Haptics = {}));
export var Device;
(function (Device) {
    /** Return information about the Android device. */
    function Info() {
        return call(DeviceInfo);
    }
    Device.Info = Info;
})(Device || (Device = {}));
export var Toast;
(function (Toast) {
    /** Show a short Android toast message. */
    function Show(message) {
        return call(ToastShow, { message });
    }
    Toast.Show = Show;
})(Toast || (Toast = {}));
