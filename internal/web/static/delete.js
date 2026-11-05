// Ask before submitting a form marked with data-confirm. The forms work
// without this script, but browsers that run it get a confirmation dialog.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (!form.matches("form[data-confirm]")) {
    return;
  }
  if (!window.confirm(form.getAttribute("data-confirm"))) {
    event.preventDefault();
  }
});
