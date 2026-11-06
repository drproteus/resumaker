// Bolds the table of contents entry whose anchor matches the URL hash. The
// links work without this script; it only adds the bold state.
(function () {
  var links = Array.from(document.querySelectorAll(".resume-toc a[href^='#']"));

  function highlight() {
    links.forEach(function (link) {
      var active = link.getAttribute("href") === window.location.hash;
      link.classList.toggle("fw-bold", active);
      if (active) {
        link.setAttribute("aria-current", "location");
      } else {
        link.removeAttribute("aria-current");
      }
    });
  }

  window.addEventListener("hashchange", highlight);
  highlight();
})();
