// lidangle - reads MacBook lid angle sensor and outputs JSON
// Build: clang -framework IOKit -framework Foundation -o lidangle lidangle.m

#import <Foundation/Foundation.h>
#import <IOKit/hid/IOHIDManager.h>

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        IOHIDManagerRef manager = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
        if (!manager) {
            printf("{\"status\":\"error\",\"msg\":\"failed to create IOHIDManager\"}\n");
            return 1;
        }

        NSDictionary *matching = @{
            @(kIOHIDVendorIDKey): @(0x05AC),
            @(kIOHIDProductIDKey): @(0x8104),
            @"PrimaryUsagePage": @(0x0020),
            @"PrimaryUsage": @(0x008A),
        };

        IOHIDManagerSetDeviceMatching(manager, (__bridge CFDictionaryRef)matching);
        IOHIDManagerOpen(manager, kIOHIDOptionsTypeNone);

        CFSetRef deviceSet = IOHIDManagerCopyDevices(manager);
        if (!deviceSet || CFSetGetCount(deviceSet) == 0) {
            printf("{\"status\":\"error\",\"msg\":\"no lid angle sensor found\"}\n");
            if (deviceSet) CFRelease(deviceSet);
            CFRelease(manager);
            return 1;
        }

        CFIndex count = CFSetGetCount(deviceSet);
        IOHIDDeviceRef *devices = calloc(count, sizeof(IOHIDDeviceRef));
        CFSetGetValues(deviceSet, (const void **)devices);

        double angle = -1;
        for (CFIndex i = 0; i < count; i++) {
            IOHIDDeviceRef device = devices[i];
            if (IOHIDDeviceOpen(device, kIOHIDOptionsTypeNone) != kIOReturnSuccess) continue;

            uint8_t report[3] = {0};
            CFIndex length = sizeof(report);
            IOReturn result = IOHIDDeviceGetReport(device, kIOHIDReportTypeFeature, 1, report, &length);

            if (result == kIOReturnSuccess && length >= 3) {
                uint16_t raw = (report[2] << 8) | report[1];
                angle = (double)raw;
            }
            IOHIDDeviceClose(device, kIOHIDOptionsTypeNone);
            if (angle >= 0) break;
        }

        free(devices);
        CFRelease(deviceSet);
        CFRelease(manager);

        if (angle >= 0) {
            printf("{\"status\":\"ok\",\"angle\":%.0f}\n", angle);
        } else {
            printf("{\"status\":\"error\",\"msg\":\"failed to read sensor\"}\n");
        }
    }
    return 0;
}
