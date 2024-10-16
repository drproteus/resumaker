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

pages/index.html:
	python pages.py
	-cp -av ./assets/css ./pages/css
	-cp -av ./assets/images ./pages/images

pages: html pdf pages/index.html
	cp build/resume.html pages/resume.html
	cp build/resume.pdf pages/resume.pdf

.PHONY: serve-pages
serve-pages: pages
	python -m http.server -d pages

.PHONY: clean
clean:
	rm -rf build/*
	rm -rf pages/*
