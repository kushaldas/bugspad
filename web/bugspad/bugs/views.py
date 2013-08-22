from django.shortcuts import render
from bugspad.bugs.forms import BugForm

def bug_create(request):
    product = request.GET.get('product')
    return render(request, 'bug_create.html', {
        'form': BugForm, 'product': product})
