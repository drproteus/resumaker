document.addEventListener("click", function (event) {
  if (event.target.closest("[data-print]")) {
    window.print();
    return;
  }

  var dismiss = event.target.closest("[data-dismiss-toolbar]");
  if (dismiss) {
    var toolbar = dismiss.closest(".print-toolbar");
    if (toolbar) {
      toolbar.hidden = true;
    }
  }
});
