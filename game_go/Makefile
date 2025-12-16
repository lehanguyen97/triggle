.PHONY: static clean

static:
	go build -buildmode=c-archive -o libgame.a .

clean:
	rm -f libgame.a libgame.h
