package register

// ReqBody defines the request body for the register endpoint.
type ReqBody struct {
	Username Username `json:"username"`
	Password Password `json:"password"`
	Referrer string   `json:"referrer"`
}

// Validate uses individual field validation logic defined in the validation.go
// file to validate requests sent the register endpoint. It returns false and
// an errors object if any of the individual validations fail.
func (r *ReqBody) Validate() *Errs {
	errs := &Errs{}

	// validate username
	if errMsg := r.Username.Validate(); errMsg != "" {
		errs.Username = errMsg
		return errs
	}

	// validate password
	if errMsg := r.Password.Validate(); errMsg != "" {
		errs.Password = errMsg
		return errs
	}

	return nil
}