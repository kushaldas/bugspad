from django.shortcuts import render
from bugspad.bugs.forms import BugForm

def bug_create(request):
    return render(request, 'bug_create.html', {'form': BugForm})
