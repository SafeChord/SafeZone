FROM python:3.13-slim

COPY toolkit/cli/command/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt --root-user-action=ignore

COPY toolkit/cli/command /app
COPY utils /app/utils

ENV PYTHONPATH=/app
WORKDIR /app

RUN chmod +x /app/main.py && ln -s /app/main.py /usr/local/bin/szcli
RUN chmod +x /app/scripts

CMD ["tail", "-f", "/dev/null"]
