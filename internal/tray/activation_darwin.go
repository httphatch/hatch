//go:build darwin

package tray

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void hatchSetActivationPolicyRegular() {
	[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	[NSApp activateIgnoringOtherApps:YES];
}

void hatchSetActivationPolicyAccessory() {
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

void hatchSetAppIcon(const void *data, int length) {
	NSData *nsdata = [NSData dataWithBytes:data length:length];
	NSImage *image = [[NSImage alloc] initWithData:nsdata];
	[NSApp setApplicationIconImage:image];
}
*/
import "C"

import "unsafe"

func setDockVisible(visible bool) {
	if visible {
		C.hatchSetActivationPolicyRegular()
	} else {
		C.hatchSetActivationPolicyAccessory()
	}
}

func setAppIcon(icon []byte) {
	if len(icon) == 0 {
		return
	}
	C.hatchSetAppIcon(unsafe.Pointer(&icon[0]), C.int(len(icon)))
}
