$(document).ready(function() {

  function updateProgressBar(pct) {
      $('#current-progress').animate({width: pct+'%'}, 400);
  }

  function updateUI() {
      $.get("/status/"+window.installID, function(resp) {
          console.log(resp);
          if (resp.status == 'waiting for ssh') {
              $('#status-line').text('Waiting for SSH…');
              updateProgressBar(15);
          } else if (resp.status == 'waiting for http') {
              $('#status-line').text('Waiting for HTTP…');
              updateProgressBar(35);
          } else if (resp.status == 'creating client token') {
              $('#status-line').text('Creating client token…');
              updateProgressBar(95);
          } else if (resp.status == 'done') {
              $('#status-line').text('Install complete');
              updateProgressBar(100);
              $('#open-dashboard').attr('href', 'http://' + resp.client_token + '@' + resp.ip_address + ':1999/dashboard');
              $('#core-info').css('display', 'block');
          } else if (resp.status == 'failed') {
              $('status-line').text('Install failed');
              updateProgressBar(0);
          }

          if (resp.status != 'done' && resp.status != 'failed') {
              setTimeout(updateUI, 1000);
          }
      });
  }

  updateUI();
});
