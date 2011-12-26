include $(GOROOT)/src/Make.inc

TARG=main
GOFILES=\
	worker.go\
	server.go\
	output.go\
	main.go\
	util.go\

include $(GOROOT)/src/Make.pkg