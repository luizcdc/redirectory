from datetime import timedelta

import django.core.validators
from django.core.exceptions import ValidationError
from django.http import HttpResponse, JsonResponse
from django.shortcuts import render
from django.core.validators import URLValidator
import httpx
import humanize

from django.conf import settings

# Create your views here.


def index(request):
    return render(request, "index.html")


def shorten(request):
    url = request.POST["url"]
    # is it valid?
    try:
        URLValidator()(url)
    except ValidationError:
        return HttpResponse("Invalid URL", status=400)
    with httpx.Client() as client:
        response = client.post(
            f"{settings.SERVICE_HOST}/set",
            json={
                "url": url,
            },
            headers={
                "Authorization": f"Bearer {settings.SERVICE_API_KEY}",
                "Content-Type": "application/json",
            },
        )
        print(response.text)
        res_json = response.json()
        # TODO: handle errors
        res_json["SERVICE_HOST"] = settings.SERVICE_HOST
        res_json["duration"] = humanize.precisedelta(
            timedelta(seconds=res_json["duration"]), minimum_unit="minutes"
        )
        print(res_json)
    return render(request, "success.html", context=res_json)
