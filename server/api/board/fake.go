package board

import "net/url"

// fakePOSTReqValidator is a test fake for POSTReqValidator.
type fakePOSTReqValidator struct {
	InReqBody POSTReqBody
	OutErr    error
}

// Validate implements the POSTReqValidator interface on fakePOSTReqValidator.
func (f *fakePOSTReqValidator) Validate(reqBody POSTReqBody) error {
	f.InReqBody = reqBody
	return f.OutErr
}

// fakeDELETEReqValidator is a test fake for DELETEReqValidator.
type fakeDELETEReqValidator struct {
	InQParams url.Values
	OutID     string
	OutErr    error
}

// Validate implements the DELETEReqValidator interface on
// fakeDELETEReqValidator.
func (f *fakeDELETEReqValidator) Validate(qParams url.Values) (string, error) {
	f.InQParams = qParams
	return f.OutID, f.OutErr
}