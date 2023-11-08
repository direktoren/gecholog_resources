# Build with
# docker build -f Dockerfile -t custom_gecholog .

FROM gecholog/gecholog:latest

# Copy the config files (remove the lines that don't apply)
COPY your_gl_config.json /app/conf/gl_config.json
COPY your_tokencounter_config.json /app/conf/tokencounter_config.json
COPY your_natstoelastic_config.json /app/conf/natstoelastic_config.json
COPY your_natstap_config.json /app/conf/natstap_config.json

# Copy license & certificates (remove the lines that don't apply)
COPY your_license.json /app/conf/license.json
COPY your_cert.pem /app/conf/cert.pem
COPY your_key.pem /app/conf/key.pem

# 5380 gecholog 4222 8222 6222 nats
EXPOSE 5380 4222 8222 6222

# Run the start script.
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]