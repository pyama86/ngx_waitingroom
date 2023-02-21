Nginx.return -> do
  begin
    v = Nginx::Var.new

    url = "/queues/#{v.host}"
    url << "/enable" if v.enable_waitingroom == "1"

    Nginx::Async::HTTP.sub_request url
    res = Nginx::Async::HTTP.last_response
    ho = Nginx::Request.new.headers_out
    ho["Set-Cookie"] = res.headers["Set-Cookie"]

    Nginx.errlogger Nginx::LOG_ERR, res.status
    case res.status
    when 200
      return Nginx::DECLINED
    when 429
      r = JSON::parse(res.body)
      %w(
        serial_no
        allowed_no
      ).each do |n|
        ho[n] = r[n].to_s
      end
      return Nginx::HTTP_SERVICE_UNAVAILABLE
    end
  rescue => e
    Nginx.errlogger Nginx::LOG_ERR, e.inspect
    Nginx.errlogger Nginx::LOG_ERR, e.backtrace.join
    return Nginx::HTTP_SERVICE_UNAVAILABLE
  end
end.call
