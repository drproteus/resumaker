// Polls a background generation and opens its result when ready. Without
// JavaScript the user can check again by reloading the page.
(function () {
  var card = document.querySelector("[data-tailoring]");
  if (!card) {
    return;
  }
  var statusURL = card.getAttribute("data-status-url");
  var errorBox = card.querySelector("[data-tailoring-error]");
  var delay = 2000;

  function fail(message) {
    errorBox.textContent = message;
    errorBox.classList.remove("d-none");
  }

  function poll() {
    fetch(statusURL, { credentials: "same-origin", cache: "no-store" })
      .then(function (response) {
        if (!response.ok) {
          throw new Error("status " + response.status);
        }
        return response.json();
      })
      .then(function (job) {
        if (job.status === "done" && job.url) {
          window.location.replace(job.url);
          return;
        }
        if (job.status === "failed") {
          window.location.reload();
          return;
        }
        setTimeout(poll, delay);
      })
      .catch(function () {
        fail("Lost contact with the server. Check again in a moment.");
        setTimeout(poll, delay * 3);
      });
  }

  poll();
})();
