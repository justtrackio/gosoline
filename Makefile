embedmd: #go install github.com/campoy/embedmd@latest
	embedmd -w $$(find . | grep "\.md")