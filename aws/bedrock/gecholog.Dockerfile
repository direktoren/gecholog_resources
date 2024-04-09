FROM gecholog/gecholog:latest
COPY ./gl_config.json /app/conf/gl_config.json
COPY ./tokencounter_config.json /app/conf/tokencounter_config.json
COPY --chmod=644 ./cert.pem /config/certs/cert.pem
COPY --chmod=644 ./key.pem /config/certs/key.pem