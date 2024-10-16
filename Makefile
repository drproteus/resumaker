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
	python app.py --pages
	-cp -av ./assets/css ./pages/css
	-cp -av ./assets/images ./pages/images

pages: pages/index.html

.PHONY: serve-pages
serve-pages:
	python -m http.server -d pages

.PHONY: clean-pages
clean-pages:
	rm -rf pages/

.PHONY: clean
clean:
	rm -f build/*
