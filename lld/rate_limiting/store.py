class Store:
	def __init__(self, max_limit=5, time_limit=5):
		self.max_limit = max_limit
		self.time_limit = time_limit
		self.limits = {}
	
	def add_entry(self, ip, timestamp):
		if ip in self.limits:
			if len(self.limits[ip]) >= self.max_limit:
				first_entry = self.limits[ip][0]
				if timestamp - first_entry <= self.time_limit:
					return { "status": "error", "message": f"Rate limited. Try again in {self.time_limit - timestamp - first_entry}" }
				
				self.limits[ip].pop(0)

			self.limits[ip].append(timestamp)
		else:
			self.limits[ip] = [timestamp]
		
		return {"status": "success", "message": "ok"}




