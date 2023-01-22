package db

// FakeCloser is a test fake for Closer.
type FakeCloser struct {
	IsCalled bool
}

// Close implements the Closer interface on FakeCloser.
func (c *FakeCloser) Close() {
	c.IsCalled = true
}

// FakeUserInserter is a test fake for Inserter[User].
type FakeUserInserter struct {
	InUser User
	OutErr error
}

// Insert implements the Inserter[User] interface on FakeUserInserter.
func (f *FakeUserInserter) Insert(user User) error {
	f.InUser = user
	return f.OutErr
}

// FakeUserSelector is a test fake for Selector[User].
type FakeUserSelector struct {
	InUserID string
	OutRes   User
	OutErr   error
}

// Select implements the Selector[User] interface on FakeUserSelector.
func (f *FakeUserSelector) Select(userID string) (User, error) {
	f.InUserID = userID
	return f.OutRes, f.OutErr
}

// FakeCounter is a test fake for Counter.
type FakeCounter struct {
	InID   string
	OutRes int
}

// Count implements the Counter interface on FakeCounter.
func (f *FakeCounter) Count(id string) int {
	f.InID = id
	return f.OutRes
}
