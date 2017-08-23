# expanderr

The expanderr (think “expander”, pronounced with a pirate accent) is a tool
which expands the Go Call Expression under your cursor to check errors. As an
example, assuming your cursor is positioned on this call expression:

```go
	os.Remove("/tmp/state.bin")
```

…invoking the expanderr will leave you with this If Statement instead:

```go
	if err := os.Remove("/tmp/state.bin"); err != nil {
		return err
	}
```

Of course, the return values match the enclosing function signature, functions
returning more than one argument are supported, and the local scope is
considered to ensure that your code still compiles.

![screencast](screencast.gif)

## Setup

Start by running `go get -u github.com/stapelberg/expanderr`. Then, follow the
section for the editor you use:

### Emacs

Add `(load "~/go/src/github.com/stapelberg/expanderr/expanderr.el")` to your Emacs configuration.

From now on, use `C-c C-e` to invoke the expanderr.

## Opportunities to contribute

* vim integration (issue #1)
* use log.Fatal if within main()
* integration for your favorite editor
* investigate support for the errors package (which one? https://github.com/pkg/errors?)

## How does this differ from goreturns?

goreturns only inserts a return statement with zero values for the current
function.

expanderr understands the signature of the call expression under your cursor and
inserts the appropriate error checking statement (including a return
statement). In practice, this eliminates the need of combining goreturns with an
editor snippet, with the additional bonus of working correctly in a larger
number of situations.
