# Build with
# docker build -f Dockerfile -t custom_gecholog .

FROM gecholog/gecholog:latest

# Copy the config files (remove the lines that don't apply)
# COPY your_ginit_config.json /app/conf/ginit_config.json
# COPY your_gui_config.json /app/conf/gui_config.json
COPY your_gl_config.json /app/conf/gl_config.json
COPY your_tokencounter_config.json /app/conf/tokencounter_config.json
COPY your_nats2log_config.json /app/conf/nats2log_config.json
COPY your_nats2file_config.json /app/conf/nats2file_config.json
COPY your_nats-server.conf /app/conf/nats-server.conf

# Copy license & certificates (remove the lines that don't apply)
COPY your_license.json /app/conf/license.json
COPY your_cert.pem /app/conf/cert.pem
COPY your_key.pem /app/conf/key.pem
COPY ca.cert /app/conf/ca.cert
