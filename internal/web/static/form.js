// Progressive enhancement for the resume editor: add, remove and reorder rows
// without a round trip. The server parses repeated fields in document order,
// so a reordered section submits in its new order. The form still works if
// this script doesn't run.
(function () {
  function showControls(root) {
    root.querySelectorAll("[data-add], [data-remove], [data-drag-handle]").forEach(function (control) {
      control.hidden = false;
    });
  }

  function addRow(kind) {
    var rows = document.querySelector('[data-rows="' + kind + '"]');
    var proto = document.querySelector('template[data-template="' + kind + '"]');
    if (!rows || !proto) {
      return;
    }
    rows.appendChild(proto.content.cloneNode(true));

    var last = rows.lastElementChild;
    if (last) {
      showControls(last);
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

  // --- Drag to reorder -----------------------------------------------------

  var drag = null;

  document.addEventListener("pointerdown", function (event) {
    var handle = event.target.closest("[data-drag-handle]");
    if (!handle || event.button !== 0 || !event.isPrimary) {
      return;
    }
    var row = handle.closest("[data-row]");
    if (!row) {
      return;
    }

    event.preventDefault();
    try {
      handle.setPointerCapture(event.pointerId);
    } catch (err) {
      // Dragging still works without capture while the pointer stays in view.
    }
    drag = {
      row: row,
      container: row.parentElement,
      handle: handle,
      pointerId: event.pointerId
    };
    row.classList.add("dragging");
    document.body.classList.add("dragging-row");
  });

  document.addEventListener("pointermove", function (event) {
    if (!drag || event.pointerId !== drag.pointerId) {
      return;
    }
    event.preventDefault();

    // The dragged row ignores pointer events, so this finds what is underneath it.
    var target = document.elementFromPoint(event.clientX, event.clientY);
    var over = target && target.closest("[data-row]");
    if (!over || over === drag.row || over.parentElement !== drag.container) {
      return;
    }

    var rect = over.getBoundingClientRect();
    if (event.clientY < rect.top + rect.height / 2) {
      drag.container.insertBefore(drag.row, over);
    } else {
      drag.container.insertBefore(drag.row, over.nextSibling);
    }
  });

  function endDrag() {
    if (!drag) {
      return;
    }
    drag.row.classList.remove("dragging");
    document.body.classList.remove("dragging-row");
    try {
      drag.handle.releasePointerCapture(drag.pointerId);
    } catch (err) {
      // Capture may already be gone; nothing to do.
    }
    drag = null;
  }

  document.addEventListener("pointerup", endDrag);
  document.addEventListener("pointercancel", endDrag);

  // Keyboard alternative: focus a handle and use the arrow keys.
  document.addEventListener("keydown", function (event) {
    if (event.key !== "ArrowUp" && event.key !== "ArrowDown") {
      return;
    }
    var handle = event.target.closest("[data-drag-handle]");
    if (!handle) {
      return;
    }
    var row = handle.closest("[data-row]");
    if (!row) {
      return;
    }
    var sibling = event.key === "ArrowUp" ? row.previousElementSibling : row.nextElementSibling;
    if (!sibling) {
      return;
    }

    event.preventDefault();
    if (event.key === "ArrowUp") {
      row.parentElement.insertBefore(row, sibling);
    } else {
      row.parentElement.insertBefore(sibling, row);
    }
    handle.focus();
  });

  // --- Click handlers ------------------------------------------------------

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

  showControls(document);
})();
