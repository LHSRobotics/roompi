$ ->
	s = new WebSocket('ws://' + location.host + '/cmd')
	s.onmessage = (m) ->
		msg = JSON.parse(m.data);
		console.log(m)
	s.onopen = () ->
		console.log("connected")
	s.onclose = () ->
		console.log("closed")
	s.onerror = () ->
		console.log("error")
	
	sendCmd = (cmd, args) ->
		msg = 
			cmd: cmd
			args: args
		s.send(JSON.stringify(msg))
	
	$('#clean').on 'click', ->
		sendCmd("clean", [])
	$('#dock').on 'click', ->
		sendCmd("dock", [])
	$('#safe').on 'click', ->
		sendCmd("safe", [])
	$('#power').on 'click', ->
		sendCmd("power", [])
	$('#start').on 'click', ->
		sendCmd("start", [])

	curb = (min, max, n) ->
		if n > max
			max 
		else if n < min
			min
		else
			n

	dragspace = 150
	maxvelo = 500
	maxrad = 2000
	move = (dx, dy) ->
		dx = curb(-dragspace, dragspace, dx)
		dy = curb(-dragspace, dragspace, dy)
			
		velocity = dy * Math.floor(maxvelo/dragspace)
		radius = dx * Math.floor(maxrad/dragspace)
			
		right = dx < 0
		radius = 2000 - Math.abs(radius)
		radius = -radius if right
			
		sendCmd("drive", [velocity, radius])
		

	$('body').on 'mousedown', (e) ->
		[ox, oy] = [e.screenX, e.screenY]
		$('body').on 'mousemove', (e) ->
			[dx, dy] = [ox - e.screenX, oy - e.screenY]
			move(dx, dy)
		$('body').on 'mouseup', (e) ->
			sendCmd("drive", [0,0])
			$('body').off('mouseup mousemove');
			
	$('body').on 'touchstart', (e) ->
		e.preventDefault()
		t = e.originalEvent.touches[0]
		
		[ox, oy] = [t.screenX, t.screenY]
		$('body').on 'touchmove', (e) ->
			t = e.originalEvent.touches[0]
			[dx, dy] = [ox - t.screenX, oy - t.screenY]
			move(dx, dy)
		$('body').on 'touchend', (e) ->
			sendCmd("drive", [0,0])
			$('body').off('touchend touchmove');

window.onHLSReady = () ->
	streamUrl = "stream.m3u8"
	
	vid = window.document["stream"]
	vid.playerLoad(streamUrl)
	vid.playerPlay()
	