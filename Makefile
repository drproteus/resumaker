all: html pdf

build/resume.html:
	mkdir -p build/
	python app.py --html

build/resume.pdf:
	mkdir -p build/
	python app.py --pdf

build/index.html: html
	cp build/resume.html build/index.html

html: build/resume.html

pdf: build/resume.pdf

pages: html pdf build/index.html

clean:
	rm -f build/*
