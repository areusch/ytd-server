function updateStatus(status) {
  var topLevel = $('#request-' + status.requestId);
  if (!topLevel)
    return;

  if (status.title) {
    topLevel.find('.name').text(status.title);
  }

  if (status.bytesTransferred > 0 && status.totalBytes > 0)
    topLevel.find('.progress').width((parseFloat(status.bytesTransferred) / parseFloat(status.totalBytes) * 100) + '%');
  
  if (status.link)
    topLevel.find('.download-link').attr('href', status.link);

  if (!status.error && status.state && status.state != 6) {
    window.setTimeout(pingRequest(status.requestId), 500);
  }
}

function pingRequest(id) {
  return function() {
    $.ajax({type: "POST",
            url: "/dl",
            data: {id: id, json: 1}}).then(updateStatus);
  };
}

function addRequest(id) {
  $('div.template').clone().attr('id', 'request-' + id).removeClass('template').appendTo($('#downloads'));
}

function handleStatus(status) {
  var id = status.requestId;
  addRequest(id);
  updateStatus(status);
}

function handleEnqueueFail(e) {
  alert('fail');
}

function enqueueURL() {
  $.ajax({type: "POST",
          url: "/dl",
          data: {url: $('#url').val(), json: 1}})
    .done(handleStatus).fail(handleEnqueueFail);
  return false;
}

function handleURLFocus() {
  if ($('#url').hasClass('empty-url')) {
    $('#url').removeClass('empty-url').val('');
  }
}

function handleURLBlur() {
  if ($('#url').val() == '') {
    $('#url').addClass('empty-url').val('Enter the URL here');
  }
}

function bindEvents() {
  $("#submit-form").submit(enqueueURL);
  $('#url').focus(handleURLFocus);
  $('#url').blur(handleURLBlur);
}

$(window).load(bindEvents);