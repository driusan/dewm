MDFILES= src/Initialize.md src/WindowManaging.md src/Keyboard.md \
	src/KeyboardWindowManagement.md src/ColumnManagement.md

all: ${MDFILES}
	rm -rf keysym
	lmt ${MDFILES}
	# Hack to work around temporary bug in lmt, where it can't handle
	# generating into subdirectory
	mv keysym keysym.go
	mkdir -p keysym
	mv keysym.go keysym/keysym.go
	go test ./...
	go install ./...
