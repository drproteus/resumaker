ifeq ($(OS),Windows_NT)
resumaker.exe:
	go build resumaker.go

resume: resumaker.exe
	./resumaker.exe data/resume.json templates/resume.gohtml assets/css/style.css

.PHONY: clean
clean:
	cmd.exe /c 'del /F /Q resumaker.exe'
	cmd.exe /c 'del /F /Q build\resume.html'
	cmd.exe /c 'del /F /Q build\resume.pdf'
else
resumaker:
	go build resumaker.go

resume: resumaker
	mkdir -p build/
	./resumaker data/resume.json templates/resume.gohtml assets/css/style.css

.PHONY: clean
clean:
	rm -rf build/*
	rm -rf resumaker
endif
