resumaker:
	go build resumaker.go

resume: resumaker
	mkdir -p build/
	./resumaker data/resume.json templates/resume.gohtml assets/css/style.css

.PHONY: clean
clean:
	rm -rf build/*
