build/resume.html:
	mkdir -p build/
	python app.py

clean:
	rm -f build/resume.html
	rm -f build/resume.pdf
