// Progressive enhancement for the resume editor: add and remove rows without a
// round trip. The server parses repeated fields, so the form still works if
// this script doesn't run.
(function () {
  function showButtons(root) {
    root.querySelectorAll("[data-add], [data-remove]").forEach(function (button) {
      button.hidden = false;
    });
  }

  function addRow(kind) {
    var rows = document.querySelector('[data-rows="' + kind + '"]');
    var proto = document.querySelector('template[data-template="' + kind + '"]');
    if (!rows || !proto) {
      return;
    }
    var row = proto.content.cloneNode(true);
    rows.appendChild(row);

    var last = rows.lastElementChild;
    showButtons(last);
    if (last) {
      last.scrollIntoView({ block: "nearest", behavior: "smooth" });
      var field = last.querySelector("input, select, textarea");
      if (field) {
        field.focus();
      }
    }
  }

  function removeRow(button) {
    var row = button.closest("[data-row]");
    if (row) {
      row.remove();
    }
  }

  document.addEventListener("click", function (event) {
    var add = event.target.closest("[data-add]");
    if (add) {
      addRow(add.getAttribute("data-add"));
      return;
    }
    var remove = event.target.closest("[data-remove]");
    if (remove) {
      removeRow(remove);
    }
  });

  // Keep the placeholder in step with the selected contact type.
  var placeholders = {
    email: "you@example.com",
    telephone: "+1 555 0100",
    website: "example.com",
    other: ""
  };
  document.addEventListener("change", function (event) {
    var select = event.target;
    if (!select.matches('select[name="contact_type"]')) {
      return;
    }
    var input = select.closest("[data-row]").querySelector('input[name="contact_value"]');
    if (input) {
      input.placeholder = placeholders[select.value] || "";
    }
  });

  showButtons(document);
})();
