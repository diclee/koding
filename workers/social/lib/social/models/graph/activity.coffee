neo4j = require "neo4j"

{Graph} = require './index'
QueryRegistry = require './queryregistry'
{race} = require "bongo"

module.exports = class Activity extends Graph

  neo4jFacets = [
    "JLink"
    "JBlogPost"
    "JTutorial"
    "JStatusUpdate"
    "JComment"
    "JOpinion"
    "JDiscussion"
    "JCodeSnip"
    "JCodeShare"
  ]

  # build facet queries
  @generateFacets:(facets)->
    facetQuery = ""
    if facets and 'Everything' not in facets
      facetQueryList = []
      for facet in facets
        return callback new KodingError "Unknown facet: " + facets.join() if facet not in neo4jFacets
        facetQueryList.push("content.name='#{facet}'")
      facetQuery = "AND (" + facetQueryList.join(' OR ') + ")"

    return facetQuery

  @generateTimeQuery:(to)->
    timeQuery = ""
    if to
      timestamp = Math.floor(to / 1000)
      timeQuery = "AND content.`meta.createdAtEpoch` < #{timestamp}"
    return timeQuery

  # generate options
  @generateOptions:(options)->
    {limit, userId, group:{groupName}} = options
    options =
      limitCount: limit or 10
      groupName : groupName
      userId    : "#{userId}"


  @fetchFolloweeContents:(options, callback)->
    requestOptions = @generateOptions options
    facet = @generateFacets options.facet
    timeQuery = @generateTimeQuery options.to
    query = QueryRegistry.activity.following facet, timeQuery
    @fetch query, requestOptions, (err, results) =>
      console.log "arguments"
      console.log arguments
      if err then return callback err
      if results? and results.length < 1 then return callback null, []
      resultData = (result.content.data for result in results)
      @objectify resultData, (objecteds)=>
        console.log "objected"
        console.log objecteds
        @getRelatedContent objecteds, options, callback



  @getRelatedContent:(results, options, callback)->
    tempRes = []
    {group:{groupName, groupId}, client} = options

    collectRelations = race (i, res, fin)=>
      id = res.id

      @fetchRelatedItems id, (err, relatedResult)=>
        if err
          return callback err
          fin()
        else
          tempRes[i].relationData =  relatedResult
          fin()
    , =>
      if groupName == "koding"
        @removePrivateContent client, groupId, tempRes, (err, cleanContent)=>
          if err then return callback err
          console.log "clean content"
          console.log cleanContent

          @revive cleanContent, (revived)=>
            console.log "revived1"
            console.log revived
            callback null, revived
      else
        @revive tempRes, (revived)=>
          console.log "revived2"
          console.log revived
          callback null, revived

    for result in results
      tempRes.push result
      collectRelations result

  @fetchRelatedItems: (itemId, callback)->
    query = """
      start koding=node:koding("id:#{itemId}")
      match koding-[r]-all
      return all, r
      order by r.createdAtEpoch DESC
      """
    @fetchRelateds query, callback

  @fetchRelateds:(query, callback)->
    @fetch query, {}, (err, results) =>
      console.log arguments
      if err then callback err
      resultData = []
      for result in results
        type = result.r.type
        data = result.all.data
        data.relationType = type
        resultData.push data

      @objectify resultData, (objected)->
        respond = {}
        for obj in objected
          type = obj.relationType
          if not respond[type] then respond[type] = []
          respond[type].push obj

        callback err, respond


  @getSecretGroups:(client, callback)->
    JGroup = require '../group'
    JGroup.some
      $or : [
        { privacy: "private" }
        { visibility: "hidden" }
      ]
      slug:
        $nin: ["koding"] # we need koding even if its private
    , {}, (err, groups)=>
      if err then return callback err
      else
        if groups.length < 1 then callback null, []
        secretGroups = []
        checkUserCanReadActivity = race (i, {client, group}, fin)=>
          group.canReadActivity client, (err, res)=>
            secretGroups.push group.slug if err
            fin()
        , -> callback null, secretGroups
        for group in groups
          checkUserCanReadActivity {client: client, group: group}

  # we may need to add public group's read permission checking
  @removePrivateContent:(client, groupId, contents, callback)->
    console.log "contents"
    console.log contents

    if contents.length < 1 then return callback null, contents
    @getSecretGroups client, (err, secretGroups)=>
      console.log "secretGroups"
      console.log err, secretGroups
      if err then return callback err
      if secretGroups.length < 1 then return callback null, contents
      filteredContent = []
      for content in contents
        filteredContent.push content if content.group not in secretGroups
      console.log "filteredContent"
      console.log filteredContent
      return callback null, filteredContent



























    ############################################################

  @fetchTagFollows:(group, to, callback)->
    {groupId, groupName} = group
    options =
      groupId   : groupId
      groupName : groupName
      to        : to

    query = QueryRegistry.bucket.newTagFollows
    @fetchFollows query, options, callback

  @fetchFollows:(query, options, callback)->
    @fetch query, options, (err, results)=>
      if err then throw err
      @generateFollows [], results, callback

  @generateFollows:(resultData, results, callback)->
    if results? and results.length < 1 then return callback null, resultData
    result = results.shift()
    data = {}
    @objectify result.follower.data, (objected)=>
      data.follower = objected
      @objectify result.r.data, (objected)=>
        data.relationship = objected
        @objectify result.followees.data, (objected)=>
          data.followee = objected
          resultData.push data
          @generateFollows resultData, results, callback
