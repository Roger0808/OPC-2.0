//go:build darwin && cgo

package securestore

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

static CFMutableDictionaryRef query(const char *profile) {
 CFMutableDictionaryRef q = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 CFStringRef account = CFStringCreateWithCString(NULL, profile, kCFStringEncodingUTF8);
 CFDictionarySetValue(q, kSecClass, kSecClassGenericPassword);
 CFDictionarySetValue(q, kSecAttrService, CFSTR("com.opc.client.session"));
 CFDictionarySetValue(q, kSecAttrAccount, account);
 CFRelease(account); return q;
}
static int load(const char *profile, void **out, long *size) {
 CFMutableDictionaryRef q = query(profile);
 CFDictionarySetValue(q, kSecReturnData, kCFBooleanTrue);
 CFDictionarySetValue(q, kSecMatchLimit, kSecMatchLimitOne);
 CFTypeRef result = NULL;
 OSStatus status = SecItemCopyMatching(q, &result); CFRelease(q);
 if(status == errSecItemNotFound) { *out = NULL; *size = 0; return 0; }
 if(status != errSecSuccess) return status;
 *size = CFDataGetLength((CFDataRef)result);
 *out = malloc(*size); memcpy(*out, CFDataGetBytePtr((CFDataRef)result), *size);
 CFRelease(result); return 0;
}
static int save(const char *profile, const void *data, long size) {
 CFMutableDictionaryRef q = query(profile);
 CFDataRef value = CFDataCreate(NULL, data, size);
 CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 CFDictionarySetValue(attrs, kSecValueData, value);
 OSStatus status = SecItemUpdate(q, attrs);
 if(status == errSecItemNotFound) {
  CFDictionarySetValue(q, kSecValueData, value);
  CFDictionarySetValue(q, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly);
  status = SecItemAdd(q, NULL);
 }
 CFRelease(value); CFRelease(attrs); CFRelease(q); return status;
}
static int removeSecret(const char *profile) {
 CFMutableDictionaryRef q = query(profile); OSStatus status = SecItemDelete(q); CFRelease(q);
 return status == errSecItemNotFound ? 0 : status;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func keychainError(status C.int) error {
	if status != 0 {
		return fmt.Errorf("macOS 钥匙串访问失败（%d）；凭据不会保存到明文文件", status)
	}
	return nil
}
func loadSecret(_ string, profile string) ([]byte, error) {
	p := C.CString(profile)
	defer C.free(unsafe.Pointer(p))
	var data unsafe.Pointer
	var size C.long
	if err := keychainError(C.load(p, &data, &size)); err != nil {
		return nil, err
	}
	defer C.free(data)
	return C.GoBytes(data, C.int(size)), nil
}
func saveSecret(_ string, profile string, data []byte) error {
	p := C.CString(profile)
	defer C.free(unsafe.Pointer(p))
	return keychainError(C.save(p, unsafe.Pointer(&data[0]), C.long(len(data))))
}
func deleteSecret(_ string, profile string) error {
	p := C.CString(profile)
	defer C.free(unsafe.Pointer(p))
	return keychainError(C.removeSecret(p))
}
