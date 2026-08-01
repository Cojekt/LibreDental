export declare namespace Haptics {
    /** Vibrate the device for the given duration in milliseconds. */
    function Vibrate(durationMs?: number): Promise<void>;
}
export declare namespace Device {
    interface Info {
        platform: string;
        manufacturer: string;
        brand: string;
        model: string;
        device: string;
        version: string;
        sdkInt: number;
    }
    /** Return information about the Android device. */
    function Info(): Promise<Info>;
}
export declare namespace Toast {
    /** Show a short Android toast message. */
    function Show(message: string): Promise<void>;
}
