/**
 * @license
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

define([
  'jquery',
  'base/js/namespace',
  'base/js/dialog',
], function($, Jupyter, dialog) {
  "use strict";

  // Default values for the configuration parameters.
  var configuration = {
    "upload_it_server_url": "http://localhost:8000/upload"
  };

  function initialize() {
    update_config();
    Jupyter.toolbar.add_buttons_group([
      Jupyter.keyboard_manager.actions.register({
  "help": "Upload notebook",
  "icon": "fa-check-square",
  "handler": show_upload_dialog
      }, "upload-notebook", "upload_it")
    ]);
  }

  function update_config() {
    // If the notebook has configuration overrides, pick them up.
    var config = Jupyter.notebook.config;
    for (var key in configuration) {
      if (config.data.hasOwnProperty(key)) {
  configuration[key] = config.data[key];
      }
    }
  }

  // Returns a JQuery object with the dialog.
  function build_upload_dialog() {
    var $upload_dialog = $("#upload_it_dialog");
    if ($upload_dialog.length > 0) {
      return $upload_dialog;
    }
    $upload_dialog = $("<div>")
      .attr("id", "upload_it_dialog");
    var $controls = $("<form>")
      .addClass("form-horizontal")
      .appendTo($upload_dialog);
    $("<div>")
      .addClass("form-group")
      .appendTo($controls)
      .append(
  $("<div>")
    .addClass("checkbox")
    .append(
      $("<label>")
        .text("Finalize submission")
        .prepend(
    $("<input>")
      .attr("type", "checkbox")
      .attr("id", "upload_it_final")
      .prop("checked", false)
        )
    )
      );
    return $upload_dialog;
  }

  function show_upload_dialog() {
    var modal = dialog.modal({
      show: false,
      title: "Upload notebook",
      notebook: Jupyter.notebook,
      keyboard_manager: Jupyter.notebook.keyboard_manager,
      body: build_upload_dialog(),
      buttons: {
  "Upload": {
    "class": "btn-primary",
    "click": function () {
      var notebook = Jupyter.notebook;
      var url = configuration.upload_it_server_url;
      var formdata = new FormData();
      var content = JSON.stringify(Jupyter.notebook.toJSON(), null, 2);
      var blob = new Blob([content], { type: "application/x-ipynb+json"});
      formdata.set("notebook", blob);
      window.console.log("Uploading ", notebook.notebook_path, " to ", url, formdata);
      $.ajax({
        url: url,
        data: formdata,
        contentType: false,
        processData: false,
        method: "POST",
        success: function(data, status, jqXHR) {
          // Open the report in a new tab.
          var u = new URL(url);
          var reportURL = u.protocol + "//" + u.host + data;
          window.open(reportURL, '_blank');
          window.console.log("Upload OK", reportURL);
        },
        error: function(jqXHR, status, err) {
          window.console.log("Upload failed", status, err);
        }
      });
    }
  },
  "done": {}
      }
    });
    modal.attr('id', 'upload_it_modal');
    modal.modal('show');
  }

  function load_jupyter_extension() {
    return Jupyter.notebook.config.loaded.then(initialize);
  }

  return {
    load_ipython_extension: load_jupyter_extension,
    load_jupyter_extension: load_jupyter_extension
  };

});
