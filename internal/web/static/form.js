// Progressive enhancement for the resume editor: add, remove and reorder rows,
// and switch between the form and JSON views. The server parses whichever view
// was submitted, so the form still works if this script doesn't run.
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

  // --- Contact placeholders ------------------------------------------------

  var placeholders = {
    email: "you@example.com",
    telephone: "+1 555 0100",
    website: "example.com",
    other: ""
  };

  function updateContactPlaceholder(row) {
    if (!row) {
      return;
    }
    var select = row.querySelector('select[name="contact_type"]');
    var input = row.querySelector('input[name="contact_value"]');
    if (select && input) {
      input.placeholder = placeholders[select.value] || "";
    }
  }

  document.addEventListener("change", function (event) {
    var select = event.target;
    if (!select.matches('select[name="contact_type"]')) {
      return;
    }
    updateContactPlaceholder(select.closest("[data-row]"));
  });

  // --- Form / JSON views ---------------------------------------------------

  var textarea = document.getElementById("data");
  var formView = document.getElementById("view-form");
  var jsonView = document.getElementById("view-json");
  var editorForm = document.querySelector(".resume-editor");
  var formPanel = document.querySelector('[data-editor-panel="form"]');
  var errorBox = document.getElementById("editor-error");

  if (!textarea || !formView || !jsonView) {
    showControls(document);
    return;
  }

  function stringValue(value) {
    return typeof value === "string" ? value : "";
  }

  function textLines(value) {
    if (typeof value === "string") {
      return value;
    }
    return Array.isArray(value) ? value.join("\n") : "";
  }

  function textList(value) {
    if (typeof value === "string") {
      return value;
    }
    return Array.isArray(value) ? value.join(", ") : "";
  }

  function fieldText(root, name) {
    var field = root.querySelector('[name="' + name + '"]');
    return field ? field.value.trim() : "";
  }

  function setField(root, name, value) {
    var field = root.querySelector('[name="' + name + '"]');
    if (field) {
      field.value = value;
    }
  }

  function sectionRows(kind) {
    return Array.from(document.querySelectorAll('[data-rows="' + kind + '"] [data-row]'));
  }

  function splitLines(value) {
    return value.split("\n").map(function (line) {
      return line.trim();
    }).filter(Boolean);
  }

  function splitList(value) {
    return value.split(",").map(function (item) {
      return item.trim();
    }).filter(Boolean);
  }

  function contactType(type) {
    return type === "email" || type === "telephone" || type === "website" ? type : "other";
  }

  function formToResume() {
    var data = {
      fullname: fieldText(document, "fullname"),
      contact: [],
      experience: [],
      education: [],
      keywords: splitList(fieldText(document, "keywords")),
      skills: []
    };

    var ai = document.querySelector('[name="ai"]');
    if (ai && ai.checked) {
      data.ai = true;
    }
    var comments = fieldText(document, "comments");
    if (comments) {
      data.comments = comments;
    }

    sectionRows("contact").forEach(function (row) {
      var value = fieldText(row, "contact_value");
      if (!value) {
        return;
      }
      data.contact.push({
        type: contactType(fieldText(row, "contact_type")),
        value: value
      });
    });

    sectionRows("experience").forEach(function (row) {
      var item = {
        title: fieldText(row, "exp_title"),
        company: fieldText(row, "exp_company"),
        location: fieldText(row, "exp_location"),
        start_date: fieldText(row, "exp_start"),
        end_date: fieldText(row, "exp_end"),
        highlights: splitLines(fieldText(row, "exp_highlights"))
      };
      if (item.title || item.company || item.location || item.start_date || item.end_date || item.highlights.length) {
        data.experience.push(item);
      }
    });

    sectionRows("education").forEach(function (row) {
      var item = {
        school: fieldText(row, "edu_school"),
        degree: fieldText(row, "edu_degree"),
        highlights: splitLines(fieldText(row, "edu_highlights"))
      };
      if (item.school || item.degree || item.highlights.length) {
        data.education.push(item);
      }
    });

    sectionRows("skill").forEach(function (row) {
      var item = {
        category: fieldText(row, "skill_category"),
        items: splitList(fieldText(row, "skill_items"))
      };
      if (item.category || item.items.length) {
        data.skills.push(item);
      }
    });

    return data;
  }

  function makeRow(kind) {
    var proto = document.querySelector('template[data-template="' + kind + '"]');
    if (!proto) {
      return null;
    }
    return proto.content.firstElementChild.cloneNode(true);
  }

  function replaceRows(kind, items, fill) {
    var container = document.querySelector('[data-rows="' + kind + '"]');
    if (!container) {
      return;
    }
    container.innerHTML = "";
    if (!items.length) {
      items = [{}];
    }
    items.forEach(function (item) {
      var row = makeRow(kind);
      if (!row) {
        return;
      }
      fill(row, item && typeof item === "object" ? item : {});
      container.appendChild(row);
      showControls(row);
    });
  }

  function populateForm(data) {
    setField(document, "fullname", stringValue(data.fullname));
    setField(document, "keywords", Array.isArray(data.keywords) ? data.keywords.join(", ") : "");
    setField(document, "comments", stringValue(data.comments));
    var ai = document.querySelector('[name="ai"]');
    if (ai) {
      ai.checked = data.ai === true;
    }

    replaceRows("contact", Array.isArray(data.contact) ? data.contact : [], function (row, item) {
      var type = contactType(stringValue(item.type));
      setField(row, "contact_type", type);
      setField(row, "contact_value", stringValue(item.value));
      updateContactPlaceholder(row);
    });

    replaceRows("experience", Array.isArray(data.experience) ? data.experience : [], function (row, item) {
      setField(row, "exp_title", stringValue(item.title));
      setField(row, "exp_company", stringValue(item.company));
      setField(row, "exp_location", stringValue(item.location));
      setField(row, "exp_start", stringValue(item.start_date));
      setField(row, "exp_end", stringValue(item.end_date));
      setField(row, "exp_highlights", textLines(item.highlights));
    });

    replaceRows("education", Array.isArray(data.education) ? data.education : [], function (row, item) {
      setField(row, "edu_school", stringValue(item.school));
      setField(row, "edu_degree", stringValue(item.degree));
      setField(row, "edu_highlights", textLines(item.highlights));
    });

    replaceRows("skill", Array.isArray(data.skills) ? data.skills : [], function (row, item) {
      setField(row, "skill_category", stringValue(item.category));
      setField(row, "skill_items", textList(item.items));
    });
  }

  function validateItems(name, items, stringFields, listFields) {
    var allowed = stringFields.concat(listFields);
    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      if (!item || typeof item !== "object" || Array.isArray(item)) {
        return name + "[" + i + "] must be an object.";
      }
      var j;
      for (j = 0; j < stringFields.length; j++) {
        if (stringFields[j] in item && typeof item[stringFields[j]] !== "string") {
          return name + "[" + i + "]." + stringFields[j] + " must be a string.";
        }
      }
      for (j = 0; j < listFields.length; j++) {
        var field = listFields[j];
        if (!(field in item)) {
          continue;
        }
        if (!Array.isArray(item[field])) {
          return name + "[" + i + "]." + field + " must be an array.";
        }
        for (var k = 0; k < item[field].length; k++) {
          if (typeof item[field][k] !== "string") {
            return name + "[" + i + "]." + field + " must contain strings.";
          }
        }
      }
      var unknown = Object.keys(item).filter(function (key) {
        return allowed.indexOf(key) < 0;
      });
      if (unknown.length) {
        return name + "[" + i + "] has unknown field(s): " + unknown.join(", ") + ".";
      }
    }
    return "";
  }

  function validateJson(data) {
    if (!data || typeof data !== "object" || Array.isArray(data)) {
      return "The JSON must be an object.";
    }
    var allowed = ["fullname", "contact", "experience", "education", "keywords", "skills", "ai", "comments"];
    var unknown = Object.keys(data).filter(function (key) {
      return allowed.indexOf(key) < 0;
    });
    if (unknown.length) {
      return "Unknown field(s): " + unknown.join(", ") + ".";
    }
    if ("fullname" in data && typeof data.fullname !== "string") {
      return "fullname must be a string.";
    }
    if ("ai" in data && typeof data.ai !== "boolean") {
      return "ai must be a boolean.";
    }
    if ("comments" in data && typeof data.comments !== "string") {
      return "comments must be a string.";
    }
    if ("keywords" in data) {
      if (!Array.isArray(data.keywords)) {
        return "keywords must be an array.";
      }
      for (var i = 0; i < data.keywords.length; i++) {
        if (typeof data.keywords[i] !== "string") {
          return "keywords must contain strings.";
        }
      }
    }

    var arrays = ["contact", "experience", "education", "skills"];
    for (var a = 0; a < arrays.length; a++) {
      if (arrays[a] in data && !Array.isArray(data[arrays[a]])) {
        return arrays[a] + " must be an array.";
      }
    }

    return validateItems("contact", data.contact || [], ["type", "value"], []) ||
      validateItems("experience", data.experience || [],
        ["title", "company", "location", "start_date", "end_date"], ["highlights"]) ||
      validateItems("education", data.education || [], ["school", "degree"], ["highlights"]) ||
      validateItems("skills", data.skills || [], ["category"], ["items"]) ||
      "";
  }

  function setEditorError(message) {
    if (!errorBox) {
      return;
    }
    errorBox.textContent = message || "";
    errorBox.classList.toggle("d-none", !message);
  }

  function setFormControlsEnabled(enabled) {
    if (!formPanel) {
      return;
    }
    formPanel.querySelectorAll("input, select, textarea").forEach(function (field) {
      field.disabled = !enabled;
    });
  }

  function setEditorView(view) {
    if (!editorForm) {
      return;
    }
    editorForm.classList.toggle("editor-view-json", view === "json");
    editorForm.classList.toggle("editor-view-form", view !== "json");
  }

  function applyInitialView() {
    if (jsonView.checked) {
      setEditorView("json");
      setFormControlsEnabled(false);
      textarea.disabled = false;
      return;
    }
    setEditorView("form");
    setFormControlsEnabled(true);
    textarea.disabled = true;
  }

  document.addEventListener("change", function (event) {
    var input = event.target;
    if (!input.matches('input[name="editor_view"]')) {
      return;
    }

    if (input.value === "json") {
      textarea.value = JSON.stringify(formToResume(), null, 2);
      textarea.disabled = false;
      setFormControlsEnabled(false);
      setEditorView("json");
      setEditorError("");
      return;
    }

    var parsed;
    try {
      parsed = JSON.parse(textarea.value);
    } catch (err) {
      setEditorError("JSON: " + err.message);
      jsonView.checked = true;
      setEditorView("json");
      return;
    }

    var problem = validateJson(parsed);
    if (problem) {
      setEditorError(problem);
      jsonView.checked = true;
      setEditorView("json");
      return;
    }

    setEditorError("");
    setFormControlsEnabled(true);
    textarea.disabled = true;
    setEditorView("form");
    populateForm(parsed);
  });

  showControls(document);
  applyInitialView();
})();
