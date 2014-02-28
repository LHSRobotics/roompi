// Generated by CoffeeScript 1.6.2
(function() {
  $(function() {
    var curb, dragspace, maxrad, maxvelo, move, s, sendCmd;

    s = new WebSocket('ws://' + location.host + '/cmd');
    s.onmessage = function(m) {
      var msg;

      msg = JSON.parse(m.data);
      return console.log(m);
    };
    s.onopen = function() {
      console.log("connected");
      sendCmd("start", []);
      return sendCmd("safe", []);
    };
    s.onclose = function() {
      return console.log("closed");
    };
    s.onerror = function() {
      return console.log("error");
    };
    sendCmd = function(cmd, args) {
      var msg;

      msg = {
        cmd: cmd,
        args: args
      };
      return s.send(JSON.stringify(msg));
    };
    curb = function(min, max, n) {
      if (n > max) {
        return max;
      } else if (n < min) {
        return min;
      } else {
        return n;
      }
    };
    dragspace = 150;
    maxvelo = 500;
    maxrad = 2000;
    move = function(dx, dy) {
      var radius, right, velocity;

      dx = curb(-dragspace, dragspace, dx);
      dy = curb(-dragspace, dragspace, dy);
      velocity = dy * Math.floor(maxvelo / dragspace);
      radius = dx * Math.floor(maxrad / dragspace);
      right = dx < 0;
      radius = 2000 - Math.abs(radius);
      if (right) {
        radius = -radius;
      }
      return sendCmd("drive", [velocity, radius]);
    };
    $('body').on('mousedown', function(e) {
      var ox, oy, _ref;

      _ref = [e.screenX, e.screenY], ox = _ref[0], oy = _ref[1];
      $('body').on('mousemove', function(e) {
        var dx, dy, _ref1;

        _ref1 = [ox - e.screenX, oy - e.screenY], dx = _ref1[0], dy = _ref1[1];
        return move(dx, dy);
      });
      return $('body').on('mouseup', function(e) {
        sendCmd("drive", [0, 0]);
        return $('body').off('mouseup mousemove');
      });
    });
    return $('body').on('touchstart', function(e) {
      var ox, oy, t, _ref;

      e.preventDefault();
      t = e.originalEvent.touches[0];
      _ref = [t.screenX, t.screenY], ox = _ref[0], oy = _ref[1];
      $('body').on('touchmove', function(e) {
        var dx, dy, _ref1;

        t = e.originalEvent.touches[0];
        _ref1 = [ox - t.screenX, oy - t.screenY], dx = _ref1[0], dy = _ref1[1];
        return move(dx, dy);
      });
      return $('body').on('touchend', function(e) {
        sendCmd("drive", [0, 0]);
        return $('body').off('touchend touchmove');
      });
    });
  });

}).call(this);
