// Polls a background resume generation and opens the new resume when it is
// ready. The page still works without this script: it shows progress and the
// user can check again.
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
