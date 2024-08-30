// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Code generated by 'go generate'; DO NOT EDIT.

// Package wsc provides access to the Windows Security Center API.
package wsc

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/dblohm7/wingoes"
	"github.com/dblohm7/wingoes/com"
	"github.com/dblohm7/wingoes/com/automation"
)

var (
	CLSID_WSCProductList = &com.CLSID{0x17072F7B, 0x9ABE, 0x4A74, [8]byte{0xA2, 0x61, 0x1E, 0xB7, 0x6B, 0x55, 0x10, 0x7A}}
)

var (
	IID_IWSCProductList = &com.IID{0x722A338C, 0x6E8E, 0x4E72, [8]byte{0xAC, 0x27, 0x14, 0x17, 0xFB, 0x0C, 0x81, 0xC2}}
	IID_IWscProduct     = &com.IID{0x8C38232E, 0x3A45, 0x4A27, [8]byte{0x92, 0xB0, 0x1A, 0x16, 0xA9, 0x75, 0xF6, 0x69}}
)

type WSC_SECURITY_PRODUCT_STATE int32

const (
	WSC_SECURITY_PRODUCT_STATE_ON      = WSC_SECURITY_PRODUCT_STATE(0)
	WSC_SECURITY_PRODUCT_STATE_OFF     = WSC_SECURITY_PRODUCT_STATE(1)
	WSC_SECURITY_PRODUCT_STATE_SNOOZED = WSC_SECURITY_PRODUCT_STATE(2)
	WSC_SECURITY_PRODUCT_STATE_EXPIRED = WSC_SECURITY_PRODUCT_STATE(3)
)

type WSC_SECURITY_SIGNATURE_STATUS int32

const (
	WSC_SECURITY_PRODUCT_OUT_OF_DATE = WSC_SECURITY_SIGNATURE_STATUS(0)
	WSC_SECURITY_PRODUCT_UP_TO_DATE  = WSC_SECURITY_SIGNATURE_STATUS(1)
)

type WSC_SECURITY_PROVIDER int32

const (
	WSC_SECURITY_PROVIDER_FIREWALL             = WSC_SECURITY_PROVIDER(1)
	WSC_SECURITY_PROVIDER_AUTOUPDATE_SETTINGS  = WSC_SECURITY_PROVIDER(2)
	WSC_SECURITY_PROVIDER_ANTIVIRUS            = WSC_SECURITY_PROVIDER(4)
	WSC_SECURITY_PROVIDER_ANTISPYWARE          = WSC_SECURITY_PROVIDER(8)
	WSC_SECURITY_PROVIDER_INTERNET_SETTINGS    = WSC_SECURITY_PROVIDER(16)
	WSC_SECURITY_PROVIDER_USER_ACCOUNT_CONTROL = WSC_SECURITY_PROVIDER(32)
	WSC_SECURITY_PROVIDER_SERVICE              = WSC_SECURITY_PROVIDER(64)
	WSC_SECURITY_PROVIDER_NONE                 = WSC_SECURITY_PROVIDER(0)
	WSC_SECURITY_PROVIDER_ALL                  = WSC_SECURITY_PROVIDER(127)
)

type SECURITY_PRODUCT_TYPE int32

const (
	SECURITY_PRODUCT_TYPE_ANTIVIRUS   = SECURITY_PRODUCT_TYPE(0)
	SECURITY_PRODUCT_TYPE_FIREWALL    = SECURITY_PRODUCT_TYPE(1)
	SECURITY_PRODUCT_TYPE_ANTISPYWARE = SECURITY_PRODUCT_TYPE(2)
)

type IWscProductABI struct {
	com.IUnknownABI // Technically IDispatch, but we're bypassing all of that atm
}

func (abi *IWscProductABI) GetProductName() (pVal string, err error) {
	var t0 automation.BSTR

	method := unsafe.Slice(abi.Vtbl, 14)[7]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&t0)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
		if e.Failed() {
			return
		}
	}

	pVal = t0.String()
	t0.Close()
	return
}

func (abi *IWscProductABI) GetProductState() (val WSC_SECURITY_PRODUCT_STATE, err error) {
	method := unsafe.Slice(abi.Vtbl, 14)[8]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&val)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
	}
	return
}

func (abi *IWscProductABI) GetSignatureStatus() (val WSC_SECURITY_SIGNATURE_STATUS, err error) {
	method := unsafe.Slice(abi.Vtbl, 14)[9]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&val)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
	}
	return
}

func (abi *IWscProductABI) GetRemediationPath() (pVal string, err error) {
	var t0 automation.BSTR

	method := unsafe.Slice(abi.Vtbl, 14)[10]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&t0)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
		if e.Failed() {
			return
		}
	}

	pVal = t0.String()
	t0.Close()
	return
}

func (abi *IWscProductABI) GetProductStateTimestamp() (pVal string, err error) {
	var t0 automation.BSTR

	method := unsafe.Slice(abi.Vtbl, 14)[11]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&t0)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
		if e.Failed() {
			return
		}
	}

	pVal = t0.String()
	t0.Close()
	return
}

func (abi *IWscProductABI) GetProductGuid() (pVal string, err error) {
	var t0 automation.BSTR

	method := unsafe.Slice(abi.Vtbl, 14)[12]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&t0)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
		if e.Failed() {
			return
		}
	}

	pVal = t0.String()
	t0.Close()
	return
}

func (abi *IWscProductABI) GetProductIsDefault() (pVal bool, err error) {
	var t0 int32

	method := unsafe.Slice(abi.Vtbl, 14)[13]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&t0)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
		if e.Failed() {
			return
		}
	}

	pVal = t0 != 0
	return
}

type WscProduct struct {
	com.GenericObject[IWscProductABI]
}

func (o WscProduct) GetProductName() (pVal string, err error) {
	p := *(o.Pp)
	return p.GetProductName()
}

func (o WscProduct) GetProductState() (val WSC_SECURITY_PRODUCT_STATE, err error) {
	p := *(o.Pp)
	return p.GetProductState()
}

func (o WscProduct) GetSignatureStatus() (val WSC_SECURITY_SIGNATURE_STATUS, err error) {
	p := *(o.Pp)
	return p.GetSignatureStatus()
}

func (o WscProduct) GetRemediationPath() (pVal string, err error) {
	p := *(o.Pp)
	return p.GetRemediationPath()
}

func (o WscProduct) GetProductStateTimestamp() (pVal string, err error) {
	p := *(o.Pp)
	return p.GetProductStateTimestamp()
}

func (o WscProduct) GetProductGuid() (pVal string, err error) {
	p := *(o.Pp)
	return p.GetProductGuid()
}

func (o WscProduct) GetProductIsDefault() (pVal bool, err error) {
	p := *(o.Pp)
	return p.GetProductIsDefault()
}

func (o WscProduct) IID() *com.IID {
	return IID_IWscProduct
}

func (o WscProduct) Make(r com.ABIReceiver) any {
	if r == nil {
		return WscProduct{}
	}

	runtime.SetFinalizer(r, com.ReleaseABI)

	pp := (**IWscProductABI)(unsafe.Pointer(r))
	return WscProduct{com.GenericObject[IWscProductABI]{Pp: pp}}
}

func (o WscProduct) MakeFromKnownABI(r **IWscProductABI) WscProduct {
	if r == nil {
		return WscProduct{}
	}

	runtime.SetFinalizer(r, func(r **IWscProductABI) { (*r).Release() })
	return WscProduct{com.GenericObject[IWscProductABI]{Pp: r}}
}

func (o WscProduct) UnsafeUnwrap() *IWscProductABI {
	return *(o.Pp)
}

type IWSCProductListABI struct {
	com.IUnknownABI // Technically IDispatch, but we're bypassing all of that atm
}

func (abi *IWSCProductListABI) Initialize(provider WSC_SECURITY_PROVIDER) (err error) {
	method := unsafe.Slice(abi.Vtbl, 10)[7]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(provider))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
	}
	return
}

func (abi *IWSCProductListABI) GetCount() (val int32, err error) {
	method := unsafe.Slice(abi.Vtbl, 10)[8]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(unsafe.Pointer(&val)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
	}
	return
}

func (abi *IWSCProductListABI) GetItem(index uint32) (val WscProduct, err error) {
	var t0 *IWscProductABI

	method := unsafe.Slice(abi.Vtbl, 10)[9]
	hr, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(abi)), uintptr(index), uintptr(unsafe.Pointer(&t0)))
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(hr)); !e.IsOK() {
		err = e
		if e.Failed() {
			return
		}
	}

	var r0 WscProduct
	val = r0.MakeFromKnownABI(&t0)
	return
}

type WSCProductList struct {
	com.GenericObject[IWSCProductListABI]
}

func (o WSCProductList) Initialize(provider WSC_SECURITY_PROVIDER) (err error) {
	p := *(o.Pp)
	return p.Initialize(provider)
}

func (o WSCProductList) GetCount() (val int32, err error) {
	p := *(o.Pp)
	return p.GetCount()
}

func (o WSCProductList) GetItem(index uint32) (val WscProduct, err error) {
	p := *(o.Pp)
	return p.GetItem(index)
}

func (o WSCProductList) IID() *com.IID {
	return IID_IWSCProductList
}

func (o WSCProductList) Make(r com.ABIReceiver) any {
	if r == nil {
		return WSCProductList{}
	}

	runtime.SetFinalizer(r, com.ReleaseABI)

	pp := (**IWSCProductListABI)(unsafe.Pointer(r))
	return WSCProductList{com.GenericObject[IWSCProductListABI]{Pp: pp}}
}

func (o WSCProductList) UnsafeUnwrap() *IWSCProductListABI {
	return *(o.Pp)
}
